package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shai/shai/internal/app"
	"github.com/shai/shai/internal/backend"
)

func runDownload(flags cliFlags) error {
	repo := flags.downloadRepo
	localName := flags.modelName

	if repo == "" {
		return runDownloadInteractive()
	}

	if localName == "" {
		localName = repoBaseName(repo)
	}

	return downloadModel(context.Background(), repo, localName, flags.revision, flags.force, false)
}

func runDownloadInteractive() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Hugging Face repo ID: ")
	repo, _ := reader.ReadString('\n')
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return fmt.Errorf("repo ID is required")
	}

	defaultName := repoBaseName(repo)
	fmt.Printf("Local model name [%s]: ", defaultName)
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)
	if name == "" {
		name = defaultName
	}

	modelsDir, err := app.ModelsDir()
	if err != nil {
		return err
	}
	dest := filepath.Join(modelsDir, name)
	if appDirNonempty(dest) {
		fmt.Printf("Destination %s already exists and is not empty. Overwrite? [y/N]: ", dest)
		answer, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(answer)) != "y" {
			return fmt.Errorf("download cancelled")
		}
	}

	return downloadModel(context.Background(), repo, name, "", true, false)
}

func downloadModel(ctx context.Context, repo, localName, revision string, force, interactive bool) error {
	modelsDir, err := app.ModelsDir()
	if err != nil {
		return err
	}
	if err := app.EnsureDirs(); err != nil {
		return err
	}

	dest := filepath.Join(modelsDir, localName)
	if appDirNonempty(dest) && !force {
		if interactive {
			return fmt.Errorf("destination exists; re-run with confirmation or use --force")
		}
		return fmt.Errorf("destination already exists: %s (use --force to overwrite)", dest)
	}

	fmt.Fprintf(os.Stderr, "Downloading %s to %s...\n", repo, dest)

	var finalPath string
	err = backend.RunDownload(ctx, backend.DownloadOpts{
		Repo:     repo,
		LocalDir: dest,
		Revision: revision,
		Force:    force,
	}, func(ev backend.Event) {
		if ev.Type == "done" {
			finalPath = ev.Path
		}
		if ev.Type == "error" {
			fmt.Fprintln(os.Stderr, ev.Message)
		}
	})
	if err != nil {
		return err
	}

	if finalPath == "" {
		finalPath = dest
	}
	fmt.Printf("Model saved to %s\n", finalPath)
	return nil
}

func repoBaseName(repo string) string {
	repo = strings.TrimSpace(repo)
	if i := strings.LastIndex(repo, "/"); i >= 0 {
		return repo[i+1:]
	}
	return repo
}

func appDirNonempty(path string) bool {
	entries, err := os.ReadDir(path)
	if err != nil || len(entries) == 0 {
		return false
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		return true
	}
	return false
}
