package backend

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/shai/shai/internal/app"
	"github.com/shai/shai/internal/version"
)

//go:embed runner.py
var runnerPy []byte

//go:embed requirements.txt
var requirementsTxt []byte

// Event is a JSONL event from the Python backend.
type Event struct {
	Type    string `json:"type"`
	Text    string `json:"text"`
	Message string `json:"message"`
	Path    string `json:"path"`
}

// Message is a chat message for generation.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// GenerateOpts are passed to the generate subcommand.
type GenerateOpts struct {
	ModelPath    string
	Device       string
	MaxTokens    int
	SystemPrompt string
	Prompt       string
	StdinContext string
	Messages     []Message
}

// DownloadOpts are passed to the download subcommand.
type DownloadOpts struct {
	Repo      string
	LocalDir  string
	Revision  string
	Force     bool
}

const backendVersionFile = "backend.version"

// EnsureInstalled writes embedded backend files and verifies the venv exists.
func EnsureInstalled() error {
	if err := app.EnsureDirs(); err != nil {
		return err
	}

	backendDir, err := app.BackendDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(backendDir, 0o755); err != nil {
		return err
	}

	versionPath := filepath.Join(backendDir, backendVersionFile)
	current := version.Version
	stored, _ := os.ReadFile(versionPath)

	runnerPath, err := app.BackendRunner()
	if err != nil {
		return err
	}
	reqPath, err := app.BackendRequirements()
	if err != nil {
		return err
	}

	needsWrite := !bytes.Equal(stored, []byte(current))
	if !needsWrite {
		if _, err := os.Stat(runnerPath); os.IsNotExist(err) {
			needsWrite = true
		}
	}

	if needsWrite {
		if err := os.WriteFile(runnerPath, runnerPy, 0o755); err != nil {
			return fmt.Errorf("write runner.py: %w", err)
		}
		if err := os.WriteFile(reqPath, requirementsTxt, 0o644); err != nil {
			return fmt.Errorf("write requirements.txt: %w", err)
		}
		if err := os.WriteFile(versionPath, []byte(current), 0o644); err != nil {
			return fmt.Errorf("write backend version: %w", err)
		}
	}

	python, err := app.VenvPython()
	if err != nil {
		return err
	}
	if _, err := os.Stat(python); os.IsNotExist(err) {
		return app.ErrVenvMissing{}
	}

	return nil
}

func pythonCmd(ctx context.Context, args ...string) (*exec.Cmd, error) {
	python, err := app.VenvPython()
	if err != nil {
		return nil, err
	}
	runner, err := app.BackendRunner()
	if err != nil {
		return nil, err
	}
	fullArgs := append([]string{runner}, args...)
	cmd := exec.CommandContext(ctx, python, fullArgs...)
	env := append(os.Environ(), "PYTHONUNBUFFERED=1")
	if libPath, err := openvinoLibPath(python); err == nil && libPath != "" {
		env = append(env, prependLibPath(libPath)...)
	}
	cmd.Env = env
	return cmd, nil
}

func prependLibPath(libPath string) []string {
	if runtime.GOOS == "darwin" {
		key := "DYLD_LIBRARY_PATH"
		existing := os.Getenv(key)
		if existing != "" {
			return []string{key + "=" + libPath + ":" + existing}
		}
		return []string{key + "=" + libPath}
	}
	key := "LD_LIBRARY_PATH"
	existing := os.Getenv(key)
	if existing != "" {
		return []string{key + "=" + libPath + ":" + existing}
	}
	return []string{key + "=" + libPath}
}

func openvinoLibPath(python string) (string, error) {
	cmd := exec.Command(python, "-c", "import openvino, os; print(os.path.join(os.path.dirname(openvino.__file__), 'libs'))")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// RunGenerate runs the generate subcommand and invokes onEvent for each JSONL line.
func RunGenerate(ctx context.Context, opts GenerateOpts, onEvent func(Event)) error {
	payload := map[string]interface{}{
		"model_path":    opts.ModelPath,
		"device":        strings.ToUpper(opts.Device),
		"max_tokens":    opts.MaxTokens,
		"system_prompt": opts.SystemPrompt,
		"prompt":        opts.Prompt,
		"stdin_context": opts.StdinContext,
		"messages":      opts.Messages,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	cmd, err := pythonCmd(ctx, "generate")
	if err != nil {
		return err
	}
	cmd.Stdin = bytes.NewReader(body)

	return runJSONL(cmd, onEvent)
}

// RunDownload runs the download subcommand.
func RunDownload(ctx context.Context, opts DownloadOpts, onEvent func(Event)) error {
	args := []string{
		"download",
		"--repo", opts.Repo,
		"--local-dir", opts.LocalDir,
	}
	if opts.Revision != "" {
		args = append(args, "--revision", opts.Revision)
	}
	if opts.Force {
		args = append(args, "--force")
	}

	cmd, err := pythonCmd(ctx, args...)
	if err != nil {
		return err
	}
	return runJSONL(cmd, onEvent)
}

func runJSONL(cmd *exec.Cmd, onEvent func(Event)) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	var stderrBuf strings.Builder
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(io.MultiWriter(os.Stderr, &stderrBuf), stderr)
	}()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start backend: %w", err)
	}

	var lastErr error
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var ev Event
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			lastErr = fmt.Errorf("parse backend output: %w", err)
			continue
		}
		if onEvent != nil {
			onEvent(ev)
		}
		if ev.Type == "error" {
			lastErr = fmt.Errorf("%s", ev.Message)
		}
	}
	if err := scanner.Err(); err != nil {
		lastErr = err
	}

	if err := cmd.Wait(); err != nil {
		wg.Wait()
		msg := strings.TrimSpace(stderrBuf.String())
		if lastErr != nil {
			return lastErr
		}
		if msg != "" {
			return fmt.Errorf("%w: %s", err, msg)
		}
		return err
	}
	wg.Wait()

	if lastErr != nil {
		return lastErr
	}
	return nil
}

// CollectGenerate runs generation and returns the final assistant text.
func CollectGenerate(ctx context.Context, opts GenerateOpts) (string, error) {
	var result string
	err := RunGenerate(ctx, opts, func(ev Event) {
		switch ev.Type {
		case "token":
			result += ev.Text
		case "done":
			if ev.Text != "" {
				result = ev.Text
			}
		}
	})
	if err != nil {
		return result, app.WrapNPUError(opts.Device, err)
	}
	return result, nil
}
