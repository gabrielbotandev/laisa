package cmd

import (
	"fmt"
	"os"

	"github.com/gabrielbotandev/laisa/internal/app"
	"github.com/gabrielbotandev/laisa/internal/version"
	"github.com/spf13/cobra"
)

var flags cliFlags

// Execute runs the root command.
func Execute() error {
	return NewRootCmd().Execute()
}

// NewRootCmd builds the laisa CLI.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "laisa [prompt]",
		Short: "Local AI Shell Assistant (OpenVINO GenAI)",
		Long: `Laisa is a terminal-first local AI assistant powered by OpenVINO GenAI.

Run without arguments to open the interactive TUI, or pass a prompt for one-shot mode.`,
		Args: cobra.ArbitraryArgs,
		RunE: runRoot,
	}

	root.Flags().BoolVar(&flags.showVersion, "version", false, "Print version")
	root.Flags().BoolVar(&flags.showConfig, "config", false, "Print config path and contents")
	root.Flags().BoolVar(&flags.listModels, "list-models", false, "List downloaded local models")
	root.Flags().BoolVar(&flags.download, "download", false, "Download a model from Hugging Face")
	root.Flags().StringVar(&flags.downloadRepo, "download-repo", "", "Hugging Face repo ID (use: laisa --download REPO via positional)")
	root.Flags().StringVar(&flags.modelName, "name", "", "Local model name for download")
	root.Flags().StringVar(&flags.revision, "revision", "", "Hugging Face revision")
	root.Flags().BoolVar(&flags.force, "force", false, "Overwrite existing model directory")
	root.Flags().StringVar(&flags.model, "model", "", "Model name or path")
	root.Flags().StringVar(&flags.device, "device", "", "OpenVINO device: CPU, NPU, or AUTO")
	root.Flags().IntVar(&flags.maxTokens, "max-tokens", 0, "Maximum generated tokens (default from config)")
	if err := root.Flags().MarkHidden("download-repo"); err != nil {
		_ = err
	}

	return root
}

func runRoot(cmd *cobra.Command, args []string) error {
	if err := app.EnsureDirs(); err != nil {
		return err
	}

	// Track whether max-tokens was explicitly set
	flags.maxTokensSet = cmd.Flags().Changed("max-tokens")

	if flags.showVersion {
		fmt.Println(version.Version)
		return nil
	}

	if flags.showConfig {
		cfg, err := app.LoadOrCreate()
		if err != nil {
			return err
		}
		out, err := cfg.FormatHuman()
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	}

	if flags.listModels {
		models, err := app.ListModels()
		if err != nil {
			return err
		}
		if len(models) == 0 {
			fmt.Println("No local models found.")
			return nil
		}
		fmt.Println("Local models:")
		for _, m := range models {
			fmt.Printf("  - %s\n", m)
		}
		return nil
	}

	// laisa --download REPO  → cobra leaves REPO in args when --download is bool
	if flags.download {
		if len(args) > 0 {
			flags.downloadRepo = args[0]
		}
		return runDownload(flags)
	}

	// One-shot: prompt arg and/or piped stdin
	hasPrompt := len(args) > 0
	hasStdin := !isTerminal(os.Stdin)

	if hasPrompt {
		if len(args) > 1 {
			return fmt.Errorf("expected a single prompt argument")
		}
		return runOneShot(flags, args[0])
	}

	if hasStdin {
		return fmt.Errorf("stdin provided but no prompt; use: cat file | laisa \"your prompt\"")
	}

	return runTUI(flags)
}
