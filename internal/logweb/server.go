package logweb

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/leo/leo-cli/internal/logview"
)

const sessionCookieName = "leo_log_session"

//go:embed assets/*
var embeddedAssets embed.FS

type Options struct {
	Now                func() time.Time
	SessionTTL         time.Duration
	FollowPollInterval time.Duration
}

type Server struct {
	catalog        *logview.Catalog
	searcher       *logview.Searcher
	follower       *logview.Follower
	now            func() time.Time
	sessionTTL     time.Duration
	bootstrapToken string
	bootstrapUsed  bool
	sessions       map[string]time.Time
	mu             sync.Mutex
	handler        http.Handler
}

func New(catalog *logview.Catalog, options Options) (*Server, error) {
	bootstrapToken, err := randomToken()
	if err != nil {
		return nil, fmt.Errorf("generate bootstrap token: %w", err)
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	ttl := options.SessionTTL
	if ttl <= 0 {
		ttl = 4 * time.Hour
	}
	follower := logview.NewFollower(catalog)
	if options.FollowPollInterval > 0 {
		follower.PollInterval = options.FollowPollInterval
	}
	server := &Server{
		catalog:        catalog,
		searcher:       logview.NewSearcher(catalog),
		follower:       follower,
		now:            now,
		sessionTTL:     ttl,
		bootstrapToken: bootstrapToken,
		sessions:       make(map[string]time.Time),
	}
	server.handler = server.routes()
	return server, nil
}

func (s *Server) BootstrapURL(baseURL string) string {
	return strings.TrimRight(baseURL, "/") + "/bootstrap?token=" + url.QueryEscape(s.bootstrapToken)
}

func (s *Server) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("X-Content-Type-Options", "nosniff")
	response.Header().Set("X-Frame-Options", "DENY")
	response.Header().Set("Referrer-Policy", "no-referrer")
	response.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src 'self'; img-src 'self' data:; style-src 'self'; script-src 'self'; base-uri 'none'; frame-ancestors 'none'")
	s.handler.ServeHTTP(response, request)
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/bootstrap", s.handleBootstrap)
	mux.Handle("/api/files", s.requireSession(http.HandlerFunc(s.handleFiles)))
	mux.Handle("/api/search", s.requireSession(http.HandlerFunc(s.handleSearch)))
	mux.Handle("/api/follow", s.requireSession(http.HandlerFunc(s.handleFollow)))
	assets, err := fs.Sub(embeddedAssets, "assets")
	if err != nil {
		panic(err)
	}
	mux.Handle("/", s.requireSession(http.FileServer(http.FS(assets))))
	return mux
}

func (s *Server) handleBootstrap(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		response.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	token := request.URL.Query().Get("token")
	s.mu.Lock()
	valid := !s.bootstrapUsed && subtle.ConstantTimeCompare([]byte(token), []byte(s.bootstrapToken)) == 1
	if valid {
		s.bootstrapUsed = true
	}
	s.mu.Unlock()
	if !valid {
		http.Error(response, "invalid or already-used bootstrap token", http.StatusUnauthorized)
		return
	}

	sessionToken, err := randomToken()
	if err != nil {
		http.Error(response, "could not create session", http.StatusInternalServerError)
		return
	}
	expires := s.now().Add(s.sessionTTL)
	s.mu.Lock()
	s.sessions[sessionToken] = expires
	s.mu.Unlock()
	http.SetCookie(response, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionToken,
		Path:     "/",
		Expires:  expires,
		MaxAge:   int(s.sessionTTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	http.Redirect(response, request, "/", http.StatusSeeOther)
}

func (s *Server) requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		cookie, err := request.Cookie(sessionCookieName)
		if err != nil {
			http.Error(response, "session required or expired", http.StatusUnauthorized)
			return
		}
		remaining, ok := s.sessionRemaining(cookie.Value)
		if !ok {
			http.Error(response, "session required or expired", http.StatusUnauthorized)
			return
		}
		ctx, cancel := context.WithTimeout(request.Context(), remaining)
		defer cancel()
		next.ServeHTTP(response, request.WithContext(ctx))
	})
}

func (s *Server) sessionRemaining(token string) (time.Duration, bool) {
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	expires, ok := s.sessions[token]
	if !ok || !now.Before(expires) {
		delete(s.sessions, token)
		return 0, false
	}
	return expires.Sub(now), true
}

func (s *Server) handleFiles(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		response.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	response.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(response).Encode(struct {
		Roots []string       `json:"roots"`
		Files []logview.File `json:"files"`
	}{Roots: s.catalog.Roots(), Files: s.catalog.Files()})
}

func (s *Server) handleSearch(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		response.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !sameOrigin(request) {
		http.Error(response, "same-origin request required", http.StatusForbidden)
		return
	}
	defer request.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(response, request.Body, 1<<20))
	decoder.DisallowUnknownFields()
	var query logview.Query
	if err := decoder.Decode(&query); err != nil {
		http.Error(response, "invalid search request: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := ensureJSONEnd(decoder); err != nil {
		http.Error(response, "invalid search request: "+err.Error(), http.StatusBadRequest)
		return
	}

	flusher, ok := response.(http.Flusher)
	if !ok {
		http.Error(response, "streaming unavailable", http.StatusInternalServerError)
		return
	}
	response.Header().Set("Content-Type", "application/x-ndjson")
	response.Header().Set("Cache-Control", "no-store")
	encoder := json.NewEncoder(response)
	wrote := false
	err := s.searcher.Search(request.Context(), query, func(event logview.Event) error {
		wrote = true
		if err := encoder.Encode(event); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	})
	if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) && !wrote {
		http.Error(response, err.Error(), http.StatusBadRequest)
	}
}

func (s *Server) handleFollow(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		response.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !validStreamOrigin(request) {
		http.Error(response, "same-origin request required", http.StatusForbidden)
		return
	}
	flusher, ok := response.(http.Flusher)
	if !ok {
		http.Error(response, "streaming unavailable", http.StatusInternalServerError)
		return
	}
	ids := splitIDs(request.URL.Query().Get("files"))
	response.Header().Set("Content-Type", "text/event-stream")
	response.Header().Set("Cache-Control", "no-store")
	response.Header().Set("Connection", "keep-alive")
	response.WriteHeader(http.StatusOK)
	flusher.Flush()
	_ = s.follower.Follow(request.Context(), ids, func(event logview.FollowEvent) error {
		data, err := json.Marshal(event)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(response, "data: %s\n\n", data); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	})
}

func sameOrigin(request *http.Request) bool {
	origin := request.Header.Get("Origin")
	if origin == "" {
		return false
	}
	scheme := "http"
	if request.TLS != nil {
		scheme = "https"
	}
	return origin == scheme+"://"+request.Host
}

func validStreamOrigin(request *http.Request) bool {
	if request.Header.Get("Origin") != "" {
		return sameOrigin(request)
	}
	return request.Header.Get("Sec-Fetch-Site") == "same-origin"
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); errors.Is(err, io.EOF) {
		return nil
	} else if err != nil {
		return err
	}
	return errors.New("multiple JSON values")
}

func splitIDs(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	ids := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			ids = append(ids, trimmed)
		}
	}
	return ids
}

func randomToken() (string, error) {
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}
