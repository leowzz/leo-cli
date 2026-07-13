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
	"path/filepath"
	"strconv"
	"strings"
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
	logRoots   []string
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
		return runLogServer(ctx, cfg, cwd, logProject, logRoots, logHost, logPort, cmd.OutOrStdout())
	},
}

type logRuntime struct {
	project   project.Selection
	catalog   *logview.Catalog
	automatic bool
}

func init() {
	logCmd.Flags().StringVar(&logProject, "project", "", "Configured project name")
	logCmd.Flags().StringArrayVar(&logRoots, "logs", nil, "Log directory (repeat for multiple roots)")
	logCmd.Flags().StringVar(&logHost, "host", "127.0.0.1", "HTTP listen host")
	logCmd.Flags().IntVar(&logPort, "port", 0, "HTTP listen port (0 chooses an available port)")
	logCmd.MarkFlagsMutuallyExclusive("project", "logs")
	rootCmd.AddCommand(logCmd)
}

func prepareLogRuntime(cfg config.Config, cwd, requestedProject string, explicitLogRoots []string) (logRuntime, []string, error) {
	if requestedProject != "" && len(explicitLogRoots) > 0 {
		return logRuntime{}, nil, errors.New("--project and --logs cannot be used together")
	}
	if requestedProject != "" {
		selection, err := project.Resolve(cwd, requestedProject, cfg.Projects)
		if err != nil {
			return logRuntime{}, nil, err
		}
		return prepareConfiguredLogRuntime(selection)
	}
	if len(explicitLogRoots) > 0 {
		return prepareAdHocLogRuntime(cwd, explicitLogRoots)
	}

	selection, err := project.Resolve(cwd, "", cfg.Projects)
	if err == nil {
		return prepareConfiguredLogRuntime(selection)
	}
	if !errors.Is(err, project.ErrNoMatch) {
		return logRuntime{}, nil, err
	}
	return prepareAdHocLogRuntime(cwd, nil)
}

func prepareConfiguredLogRuntime(selection project.Selection) (logRuntime, []string, error) {
	runtime, warnings, err := buildLogRuntime(selection, false, nil)
	if err != nil {
		return logRuntime{}, warnings, configuredLogRuntimeError(selection, warnings, err)
	}
	return runtime, warnings, nil
}

func buildLogRuntime(selection project.Selection, automatic bool, initialWarnings []string) (logRuntime, []string, error) {
	catalog, warnings, err := logview.BuildCatalog(selection.Root, selection.Config.Logs)
	warnings = append(append([]string(nil), initialWarnings...), warnings...)
	if err != nil {
		return logRuntime{}, warnings, err
	}
	if len(catalog.Files()) == 0 {
		return logRuntime{}, warnings, errors.New("no eligible log files")
	}
	return logRuntime{project: selection, catalog: catalog, automatic: automatic}, warnings, nil
}

func prepareAdHocLogRuntime(cwd string, explicitLogRoots []string) (logRuntime, []string, error) {
	root, err := project.FindRoot(cwd)
	if err != nil {
		return logRuntime{}, nil, err
	}
	var roots []string
	var warnings []string
	if len(explicitLogRoots) > 0 {
		roots, err = expandExplicitLogRoots(cwd, explicitLogRoots)
		if err != nil {
			return logRuntime{}, nil, err
		}
	} else {
		roots, warnings = discoverLogRoots(root)
	}
	if len(roots) == 0 {
		if len(explicitLogRoots) > 0 {
			return logRuntime{}, warnings, explicitLogRootsError(explicitLogRoots, warnings)
		}
		return logRuntime{}, warnings, autoLogDiscoveryError(root, warnings)
	}
	selection := project.Selection{
		Name:   filepath.Base(root),
		Root:   root,
		Config: config.ProjectConfig{Logs: roots},
	}
	runtime, catalogWarnings, err := buildLogRuntime(selection, true, warnings)
	if err != nil {
		if len(explicitLogRoots) > 0 {
			return logRuntime{}, catalogWarnings, explicitLogRootsError(explicitLogRoots, catalogWarnings)
		}
		return logRuntime{}, catalogWarnings, autoLogDiscoveryError(root, catalogWarnings)
	}
	return runtime, catalogWarnings, nil
}

func configuredLogRuntimeError(selection project.Selection, warnings []string, cause error) error {
	roots := "(none)"
	if len(selection.Config.Logs) > 0 {
		roots = strings.Join(selection.Config.Logs, ", ")
	}
	return fmt.Errorf(
		"no log files found for configured project %q at %s\n"+
			"configured log directories: %s\n"+
			"cause: %w%s",
		selection.Name,
		selection.Root,
		roots,
		cause,
		formatLogWarnings(warnings),
	)
}

func autoLogDiscoveryError(root string, warnings []string) error {
	return fmt.Errorf(
		"no log files found for %s\n"+
			"tried common log directories and log/logs folders up to depth 4\n"+
			"run: leo log --logs ./path/to/logs\n"+
			"or add:\n  proj:\n    %s:\n      logs:\n        - runtime/logs%s",
		root,
		filepath.Base(root),
		formatLogWarnings(warnings),
	)
}

func explicitLogRootsError(roots, warnings []string) error {
	return fmt.Errorf(
		"no log files found for --logs %s; correct the supplied log directories%s",
		strings.Join(roots, ", "),
		formatLogWarnings(warnings),
	)
}

func formatLogWarnings(warnings []string) string {
	if len(warnings) == 0 {
		return ""
	}
	return "\nwarnings:\n  " + strings.Join(warnings, "\n  ")
}

func runLogServer(ctx context.Context, cfg config.Config, cwd, requestedProject string, explicitLogRoots []string, host string, port int, stdout io.Writer) error {
	runtime, warnings, err := prepareLogRuntime(cfg, cwd, requestedProject, explicitLogRoots)
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
	name := runtime.project.Name
	if runtime.automatic {
		name += " (auto)"
	}
	fmt.Fprintf(stdout, "Project: %s\n", name)
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
