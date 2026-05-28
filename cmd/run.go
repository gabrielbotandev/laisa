package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gabrielbotandev/laisa/internal/app"
	"github.com/gabrielbotandev/laisa/internal/backend"
	"github.com/gabrielbotandev/laisa/internal/tui"
)

func runOneShot(flags cliFlags, prompt string) error {
	cfg, err := app.LoadOrCreate()
	if err != nil {
		return err
	}
	if err := backend.EnsureInstalled(); err != nil {
		return err
	}

	opts := app.EffectiveRunOptions(cfg, flags.model, flags.device, flags.maxTokens, flags.maxTokensSet)
	modelPath, err := app.ResolveModelOrDefault(flags.model, cfg)
	if err != nil {
		return err
	}
	opts.Model = modelPath

	var stdinContext string
	if !isTerminal(os.Stdin) {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
		stdinContext = strings.TrimSpace(string(data))
	}

	if prompt == "" && stdinContext == "" {
		return fmt.Errorf("prompt is required")
	}

	genOpts := backend.GenerateOpts{
		ModelPath:    modelPath,
		Device:       opts.Device,
		MaxTokens:    opts.MaxTokens,
		SystemPrompt: opts.SystemPrompt,
		Prompt:       prompt,
		StdinContext: stdinContext,
	}

	stdoutTTY := isTerminal(os.Stdout)
	var full strings.Builder

	err = backend.RunGenerate(context.Background(), genOpts, func(ev backend.Event) {
		switch ev.Type {
		case "token":
			full.WriteString(ev.Text)
			if stdoutTTY {
				fmt.Print(ev.Text)
			}
		case "done":
			if ev.Text != "" {
				full.Reset()
				full.WriteString(ev.Text)
			}
		case "error":
			fmt.Fprintln(os.Stderr, ev.Message)
		}
	})
	if err != nil {
		return app.WrapNPUError(opts.Device, err)
	}

	if stdoutTTY && full.Len() > 0 {
		fmt.Println()
	} else if !stdoutTTY {
		fmt.Println(strings.TrimSpace(full.String()))
	}

	return nil
}

func runTUI(flags cliFlags) error {
	cfg, err := app.LoadOrCreate()
	if err != nil {
		return err
	}
	if err := backend.EnsureInstalled(); err != nil {
		return err
	}

	opts := app.EffectiveRunOptions(cfg, flags.model, flags.device, flags.maxTokens, flags.maxTokensSet)
	modelName := flags.model
	if modelName == "" {
		modelName = cfg.DefaultModel
	}

	program := tui.NewProgram(cfg, opts, modelName)
	if _, err := program.Run(); err != nil {
		return err
	}
	return nil
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
