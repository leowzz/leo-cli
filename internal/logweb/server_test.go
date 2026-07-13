package logweb

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/leo/leo-cli/internal/logview"
)

func TestBootstrapExchangesOneTimeTokenForCleanSession(t *testing.T) {
	server := newTestServer(t, nil)
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()
	client := &http.Client{CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}

	response, err := client.Get(server.BootstrapURL(httpServer.URL))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusSeeOther || response.Header.Get("Location") != "/" {
		t.Fatalf("bootstrap status/location = %d %q", response.StatusCode, response.Header.Get("Location"))
	}
	cookies := response.Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode {
		t.Fatalf("cookies = %#v", cookies)
	}
	if strings.Contains(response.Header.Get("Location"), "token") {
		t.Fatalf("redirect leaked token: %q", response.Header.Get("Location"))
	}

	reused, err := client.Get(server.BootstrapURL(httpServer.URL))
	if err != nil {
		t.Fatal(err)
	}
	reused.Body.Close()
	if reused.StatusCode != http.StatusUnauthorized {
		t.Fatalf("reused token status = %d, want 401", reused.StatusCode)
	}

	request, _ := http.NewRequest(http.MethodGet, httpServer.URL+"/", nil)
	request.AddCookie(cookies[0])
	page, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(page.Body)
	page.Body.Close()
	if page.StatusCode != http.StatusOK || !bytes.Contains(body, []byte("Log Viewer")) {
		t.Fatalf("page status/body = %d %q", page.StatusCode, body)
	}
	if got := page.Header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q", got)
	}
}

func TestAPIsRequireUnexpiredSession(t *testing.T) {
	now := time.Date(2026, 7, 11, 10, 0, 0, 0, time.UTC)
	server := newTestServer(t, &Options{Now: func() time.Time { return now }, SessionTTL: time.Minute})
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()

	unauthorized, err := http.Get(httpServer.URL + "/api/files")
	if err != nil {
		t.Fatal(err)
	}
	unauthorized.Body.Close()
	if unauthorized.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d", unauthorized.StatusCode)
	}

	cookie := bootstrapCookie(t, server, httpServer.URL)
	request, _ := http.NewRequest(http.MethodGet, httpServer.URL+"/api/files", nil)
	request.AddCookie(cookie)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Files []logview.File `json:"files"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK || len(payload.Files) != 1 {
		t.Fatalf("files status/payload = %d %#v", response.StatusCode, payload)
	}

	now = now.Add(2 * time.Minute)
	expired, _ := http.NewRequest(http.MethodGet, httpServer.URL+"/api/files", nil)
	expired.AddCookie(cookie)
	expiredResponse, err := http.DefaultClient.Do(expired)
	if err != nil {
		t.Fatal(err)
	}
	expiredResponse.Body.Close()
	if expiredResponse.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expired status = %d, want 401", expiredResponse.StatusCode)
	}
}

func TestSearchRequiresSameOriginAndStreamsNDJSON(t *testing.T) {
	server := newTestServer(t, nil)
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()
	cookie := bootstrapCookie(t, server, httpServer.URL)
	body := `{"include":["match"],"includeUnparsed":true}`

	for _, origin := range []string{"", "https://evil.example"} {
		request, _ := http.NewRequest(http.MethodPost, httpServer.URL+"/api/search", strings.NewReader(body))
		request.AddCookie(cookie)
		if origin != "" {
			request.Header.Set("Origin", origin)
		}
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusForbidden {
			t.Fatalf("origin %q status = %d, want 403", origin, response.StatusCode)
		}
	}

	request, _ := http.NewRequest(http.MethodPost, httpServer.URL+"/api/search", strings.NewReader(body))
	request.AddCookie(cookie)
	request.Header.Set("Origin", httpServer.URL)
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.Header.Get("Content-Type") != "application/x-ndjson" {
		t.Fatalf("search status/content-type = %d %q", response.StatusCode, response.Header.Get("Content-Type"))
	}
	decoder := json.NewDecoder(response.Body)
	var types []string
	for decoder.More() {
		var event logview.Event
		if err := decoder.Decode(&event); err != nil {
			t.Fatal(err)
		}
		types = append(types, event.Type)
	}
	if !contains(types, "result") || types[len(types)-1] != "done" {
		t.Fatalf("event types = %v", types)
	}
}

func TestFollowStreamsSSE(t *testing.T) {
	server := newTestServer(t, nil)
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()
	cookie := bootstrapCookie(t, server, httpServer.URL)
	fileID := server.catalog.Files()[0].ID
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, httpServer.URL+"/api/follow?files="+fileID, nil)
	request.AddCookie(cookie)
	request.Header.Set("Sec-Fetch-Site", "same-origin")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.HasPrefix(response.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("follow status/content-type = %d %q", response.StatusCode, response.Header.Get("Content-Type"))
	}
	line, err := bufio.NewReader(response.Body).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(line, "data: ") || !strings.Contains(line, "match") {
		t.Fatalf("SSE line = %q", line)
	}
}

func TestFollowRejectsRequestWithoutOriginSignals(t *testing.T) {
	server := newTestServer(t, nil)
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()
	cookie := bootstrapCookie(t, server, httpServer.URL)
	request, _ := http.NewRequest(http.MethodGet, httpServer.URL+"/api/follow", nil)
	request.AddCookie(cookie)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("follow status = %d, want 403", response.StatusCode)
	}
}

func TestSessionMiddlewareCancelsActiveRequestAtExpiry(t *testing.T) {
	server := newTestServer(t, nil)
	token := "expiring-session"
	server.mu.Lock()
	server.sessions[token] = time.Now().Add(30 * time.Millisecond)
	server.mu.Unlock()
	handler := server.requireSession(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		select {
		case <-request.Context().Done():
			response.WriteHeader(http.StatusNoContent)
		case <-time.After(250 * time.Millisecond):
			http.Error(response, "request context did not expire", http.StatusInternalServerError)
		}
	}))
	request := httptest.NewRequest(http.MethodGet, "/stream", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("active request status = %d, want 204", response.Code)
	}
}

func TestWorkspaceContainsOperationalControls(t *testing.T) {
	server := newTestServer(t, nil)
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()
	cookie := bootstrapCookie(t, server, httpServer.URL)
	request, _ := http.NewRequest(http.MethodGet, httpServer.URL+"/", nil)
	request.AddCookie(cookie)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		`id="file-tree"`,
		`role="group"`,
		`aria-labelledby="range-label"`,
		`id="range-apply"`,
		`id="range-menu"`,
		`value="60"`,
		`value="300"`,
		`value="600"`,
		`id="include"`,
		`id="exclude"`,
		`id="search"`,
		`id="cancel"`,
		`id="clear"`,
		`id="show-unparsed"`,
		`<input id="show-unparsed" type="checkbox">`,
		`id="log-body"`,
		`id="follow"`,
		`id="jump-latest"`,
	} {
		if !bytes.Contains(body, []byte(required)) {
			t.Errorf("workspace is missing %s", required)
		}
	}
}

func TestWorkspaceContainsResizableTableAndActionMenu(t *testing.T) {
	server := newTestServer(t, nil)
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()
	cookie := bootstrapCookie(t, server, httpServer.URL)

	requiredByPath := map[string][]string{
		"/": {
			`id="log-table"`,
			`data-column="time"`,
			`data-column="level"`,
			`data-column="search-id"`,
			`data-column="user-id"`,
			`data-column="source"`,
			`data-column="message"`,
			`class="column-resize"`,
			`id="cell-action-menu" class="cell-action-menu" role="menu" hidden`,
		},
		"/app.css": {
			`.column-resize`,
			`col-resize`,
			`.cell-action-trigger`,
			`.message-text`,
			`.message-row.expanded`,
			`.cell-action-menu`,
		},
		"/app.js": {
			`initColumnResizing()`,
			`actionMenuSession: 0`,
			`const actionMenuSession = state.actionMenuSession`,
			`state.actionMenuSession !== actionMenuSession`,
			`scheduleMessageDisclosureUpdate`,
			`lostpointercapture`,
			`ArrowLeft`,
			`openCellActionMenu`,
			`closeCellActionMenu`,
			`focusout`,
			`setTimeout`,
			`updateMessageDisclosure`,
			`copyText`,
			`window.isSecureContext`,
			`if (state.actionMenuTrigger !== trigger) return`,
			`const previousActiveElement = document.activeElement`,
			`selection.getRangeAt(index).cloneRange()`,
			`selection.addRange(range)`,
		},
	}

	for path, required := range requiredByPath {
		request, _ := http.NewRequest(http.MethodGet, httpServer.URL+path, nil)
		request.AddCookie(cookie)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusOK {
			t.Fatalf("GET %s status = %d, want 200", path, response.StatusCode)
		}
		for _, marker := range required {
			if !bytes.Contains(body, []byte(marker)) {
				t.Errorf("GET %s is missing %q", path, marker)
			}
		}
	}
}

func TestWorkspaceScriptLetsPointerClickFinishBeforeFocusoutClose(t *testing.T) {
	body, err := embeddedAssets.ReadFile("assets/app.js")
	if err != nil {
		t.Fatal(err)
	}
	focusoutHandler := `elements.cellActionMenu.addEventListener("focusout", () => {
  setTimeout(() => {`
	if !bytes.Contains(body, []byte(focusoutHandler)) {
		t.Fatal("action menu focusout closes before a pointer click on another menu item can finish")
	}
}

func TestWorkspaceScriptGuardsStaleSearchAndLiveOrder(t *testing.T) {
	server := newTestServer(t, nil)
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()
	cookie := bootstrapCookie(t, server, httpServer.URL)
	request, _ := http.NewRequest(http.MethodGet, httpServer.URL+"/app.js", nil)
	request.AddCookie(cookie)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		`state.searchController !== controller`,
		`row.dataset.live = live ? "true" : "false"`,
		`insertHistoricalRow(row)`,
		`row.dataset.live !== "true" && row.dataset.timestamp`,
	} {
		if !bytes.Contains(body, []byte(required)) {
			t.Errorf("workspace script is missing %q", required)
		}
	}
}

func TestWorkspaceScriptStartsFollowAfterCatalog(t *testing.T) {
	server := newTestServer(t, nil)
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()
	cookie := bootstrapCookie(t, server, httpServer.URL)
	request, _ := http.NewRequest(http.MethodGet, httpServer.URL+"/app.js", nil)
	request.AddCookie(cookie)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(body, []byte("runSearch();\n    startFollow();")) {
		t.Fatal("workspace does not start Follow after the initial search")
	}
}

func TestWorkspaceScriptSupportsUnparsedFilter(t *testing.T) {
	server := newTestServer(t, nil)
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()
	cookie := bootstrapCookie(t, server, httpServer.URL)
	request, _ := http.NewRequest(http.MethodGet, httpServer.URL+"/app.js", nil)
	request.AddCookie(cookie)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		`includeUnparsed: elements.showUnparsed.checked`,
		`elements.showUnparsed.addEventListener("change", runSearch)`,
		`if (!record.parsed && !elements.showUnparsed.checked) return`,
	} {
		if !bytes.Contains(body, []byte(required)) {
			t.Errorf("workspace script is missing %q", required)
		}
	}
}

func newTestServer(t *testing.T, options *Options) *Server {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "app.log"), []byte("match\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	catalog, _, err := logview.BuildCatalog(root, []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	if options == nil {
		options = &Options{}
	}
	options.FollowPollInterval = 5 * time.Millisecond
	server, err := New(catalog, *options)
	if err != nil {
		t.Fatal(err)
	}
	return server
}

func bootstrapCookie(t *testing.T, server *Server, baseURL string) *http.Cookie {
	t.Helper()
	client := &http.Client{CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Get(server.BootstrapURL(baseURL))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if len(response.Cookies()) != 1 {
		t.Fatalf("bootstrap cookies = %#v", response.Cookies())
	}
	return response.Cookies()[0]
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
