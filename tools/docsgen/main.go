package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/leo/leo-cli/cmd"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

type localeOutput struct {
	dir         string
	description string
	note        string
}

func main() {
	if err := generate(cmd.RootCommand(), filepath.Join("site", "src", "content", "docs")); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate(root *cobra.Command, contentRoot string) error {
	root.InitDefaultHelpCmd()
	root.InitDefaultCompletionCmd()
	root.DisableAutoGenTag = true

	locales := []localeOutput{
		{
			description: "由 leo CLI 命令树生成的命令参考。",
			note:        "> 本页由 Cobra 命令树生成；命令、参数和输出保持 CLI 原文。",
		},
		{
			dir:         "en",
			description: "Command reference generated from the leo CLI command tree.",
			note:        "> This page is generated from the Cobra command tree.",
		},
	}

	for _, locale := range locales {
		outputDir := filepath.Join(contentRoot, locale.dir, "reference", "commands")
		if err := os.RemoveAll(outputDir); err != nil {
			return err
		}
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			return err
		}

		prepend := func(filename string) string {
			base := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
			title := strings.ReplaceAll(base, "_", " ")
			return fmt.Sprintf("---\ntitle: %q\ndescription: %q\n---\n\n%s\n\n", title, locale.description, locale.note)
		}
		link := func(name string) string {
			stem := strings.TrimSuffix(name, filepath.Ext(name))
			return "../" + filepath.ToSlash(stem) + "/"
		}
		if err := doc.GenMarkdownTreeCustom(root, outputDir, prepend, link); err != nil {
			return err
		}
	}
	return nil
}
