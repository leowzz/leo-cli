package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/leo/leo-cli/internal/config"
	"github.com/leo/leo-cli/internal/logview"
	"github.com/leo/leo-cli/internal/logweb"
	"github.com/leo/leo-cli/internal/project"
	"github.com/spf13/cobra"
)

var (
	logProject string
	logHost    string
	logPort    int
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Open a temporary browser workspace for project logs",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		ctx, stop := signal.NotifyContext(commandContext(cmd), os.Interrupt, syscall.SIGTERM)
		defer stop()
		return runLogServer(ctx, cfg, cwd, logProject, logHost, logPort, cmd.OutOrStdout())
	},
}

type logRuntime struct {
	project project.Selection
	catalog *logview.Catalog
}

func init() {
	logCmd.Flags().StringVar(&logProject, "project", "", "Configured project name")
	logCmd.Flags().StringVar(&logHost, "host", "127.0.0.1", "HTTP listen host")
	logCmd.Flags().IntVar(&logPort, "port", 0, "HTTP listen port (0 chooses an available port)")
	rootCmd.AddCommand(logCmd)
}

func prepareLogRuntime(cfg config.Config, cwd, requestedProject string) (logRuntime, []string, error) {
	selection, err := project.Resolve(cwd, requestedProject, cfg.Projects)
	if err != nil {
		return logRuntime{}, nil, err
	}
	catalog, warnings, err := logview.BuildCatalog(selection.Root, selection.Config.Logs)
	if err != nil {
		return logRuntime{}, warnings, err
	}
	return logRuntime{project: selection, catalog: catalog}, warnings, nil
}

func runLogServer(ctx context.Context, cfg config.Config, cwd, requestedProject, host string, port int, stdout io.Writer) error {
	runtime, warnings, err := prepareLogRuntime(cfg, cwd, requestedProject)
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return fmt.Errorf("listen for log viewer: %w", err)
	}
	defer listener.Close()

	web, err := logweb.New(runtime.catalog, logweb.Options{})
	if err != nil {
		return err
	}
	address, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return fmt.Errorf("unexpected listener address %q", listener.Addr())
	}
	publicHost, err := advertisedLogHost(host, os.Hostname)
	if err != nil {
		return err
	}
	baseURL := "http://" + net.JoinHostPort(publicHost, strconv.Itoa(address.Port))
	printLogStartup(stdout, runtime, warnings, web.BootstrapURL(baseURL))

	httpServer := &http.Server{
		Handler:           web,
		ReadHeaderTimeout: 5 * time.Second,
	}
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- httpServer.Serve(listener)
	}()

	select {
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shut down log viewer: %w", err)
		}
		err := <-serveErr
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}

func advertisedLogHost(bindHost string, hostname func() (string, error)) (string, error) {
	if bindHost == "" {
		return "127.0.0.1", nil
	}
	if address := net.ParseIP(bindHost); address == nil || !address.IsUnspecified() {
		return bindHost, nil
	}
	name, err := hostname()
	if err != nil {
		return "", fmt.Errorf("determine advertised log viewer host: %w", err)
	}
	if name == "" {
		return "", errors.New("determine advertised log viewer host: empty hostname")
	}
	return name, nil
}

func printLogStartup(stdout io.Writer, runtime logRuntime, warnings []string, bootstrapURL string) {
	fmt.Fprintf(stdout, "Project: %s\n", runtime.project.Name)
	fmt.Fprintf(stdout, "Root: %s\n", runtime.project.Root)
	fmt.Fprintln(stdout, "Logs:")
	for _, root := range runtime.catalog.Roots() {
		fmt.Fprintf(stdout, "  %s\n", root)
	}
	for _, warning := range warnings {
		fmt.Fprintf(stdout, "Warning: %s\n", warning)
	}
	fmt.Fprintf(stdout, "Open: %s\n", bootstrapURL)
}
