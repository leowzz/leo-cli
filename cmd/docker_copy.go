package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/leo/leo-cli/internal/config"
	"github.com/leo/leo-cli/internal/dockercopy"
	"github.com/spf13/cobra"
)

type commandRunner func(context.Context, string, []string, io.Writer, io.Writer) error

var dockerCopyDryRun bool
var dockerCopyPlatform string

const defaultDockerCopyPlatform = "linux/amd64"

var dockerCmd = &cobra.Command{
	Use:   "docker",
	Short: "Docker registry helpers",
}

var dockerCopyCmd = &cobra.Command{
	Use:   "copy SOURCE DESTINATION",
	Short: "Copy a Docker image between configured registries",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		return runDockerCopy(commandContext(cmd), cfg, args[0], args[1], dockerCopyPlatform, dockerCopyDryRun, cmd.OutOrStdout(), cmd.ErrOrStderr(), execCommand)
	},
}

var dockerListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured Docker registry aliases",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		return runDockerList(cfg, cmd.OutOrStdout())
	},
}

func init() {
	dockerCopyCmd.Flags().BoolVar(&dockerCopyDryRun, "dry", false, "Print the skopeo command without running it")
	dockerCopyCmd.Flags().StringVar(&dockerCopyPlatform, "platform", defaultDockerCopyPlatform, "Platform to copy (OS/ARCH or OS/ARCH/VARIANT)")
	rootCmd.AddCommand(dockerCmd)
	dockerCmd.AddCommand(dockerCopyCmd)
	dockerCmd.AddCommand(dockerListCmd)
}

func runDockerCopy(ctx context.Context, cfg config.Config, source, destination, platform string, dryRun bool, stdout, stderr io.Writer, runner commandRunner) error {
	copySpec, err := dockercopy.Resolve(cfg.Docker.Registries, source, destination)
	if err != nil {
		return err
	}

	args, err := skopeoCopyArgs(copySpec.Source, copySpec.Destination, platform)
	if err != nil {
		return err
	}
	if dryRun {
		fmt.Fprintln(stdout, renderCommand("skopeo", args))
		return nil
	}

	return runner(ctx, "skopeo", args, stdout, stderr)
}

type copyPlatform struct {
	os      string
	arch    string
	variant string
}

func parseCopyPlatform(value string) (copyPlatform, error) {
	parts := strings.Split(value, "/")
	if len(parts) < 2 || len(parts) > 3 || parts[0] == "" || parts[1] == "" {
		return copyPlatform{}, fmt.Errorf("platform must be OS/ARCH or OS/ARCH/VARIANT, got %q", value)
	}
	platform := copyPlatform{os: parts[0], arch: parts[1]}
	if len(parts) == 3 {
		if parts[2] == "" {
			return copyPlatform{}, fmt.Errorf("platform must be OS/ARCH or OS/ARCH/VARIANT, got %q", value)
		}
		platform.variant = parts[2]
	}
	return platform, nil
}

func skopeoCopyArgs(source, destination, platform string) ([]string, error) {
	parsed, err := parseCopyPlatform(platform)
	if err != nil {
		return nil, err
	}

	args := []string{"copy", "--override-os", parsed.os, "--override-arch", parsed.arch}
	if parsed.variant != "" {
		args = append(args, "--override-variant", parsed.variant)
	}
	args = append(args, "docker://"+source, "docker://"+destination)
	return args, nil
}

func renderCommand(name string, args []string) string {
	if len(args) == 0 {
		return name
	}
	return name + " " + strings.Join(args, " ")
}

func runDockerList(cfg config.Config, stdout io.Writer) error {
	if len(cfg.Docker.Registries) == 0 {
		_, err := fmt.Fprintln(stdout, "No Docker registries configured.")
		return err
	}

	aliases := make([]string, 0, len(cfg.Docker.Registries))
	for alias := range cfg.Docker.Registries {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)

	for _, alias := range aliases {
		if _, err := fmt.Fprintf(stdout, "%s\t%s\n", alias, cfg.Docker.Registries[alias]); err != nil {
			return err
		}
	}
	return nil
}

func execCommand(ctx context.Context, name string, args []string, stdout, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = os.Environ()
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
