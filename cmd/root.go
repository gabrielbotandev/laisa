package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Execute runs the root command.
func Execute() error {
	return NewRootCmd().Execute()
}

// NewRootCmd builds the shai CLI.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "shai",
		Short: "Local OpenVINO GenAI assistant",
		RunE:  runRoot,
	}
	registerFlags(root)
	return root
}

func registerFlags(cmd *cobra.Command) {
	// Flags registered in run.go after full implementation
	_ = cmd
}

func runRoot(cmd *cobra.Command, args []string) error {
	_ = cmd
	_ = args
	return fmt.Errorf("not implemented")
}

func exitErr(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
