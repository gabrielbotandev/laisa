package app

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ListModels returns names of non-empty model directories.
func ListModels() ([]string, error) {
	dir, err := ModelsDir()
	if err != nil {
		return nil, err
	}
	if err := EnsureDirs(); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		p := filepath.Join(dir, e.Name())
		if dirHasFiles(p) {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func dirHasFiles(path string) bool {
	entries, err := os.ReadDir(path)
	if err != nil || len(entries) == 0 {
		return false
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		return true
	}
	return false
}

// ResolveModel resolves a model name or filesystem path.
func ResolveModel(nameOrPath string) (string, error) {
	if nameOrPath == "" {
		return "", ErrNoModel{}
	}

	// Absolute or relative existing path
	if filepath.IsAbs(nameOrPath) {
		if info, err := os.Stat(nameOrPath); err == nil && info.IsDir() {
			return filepath.Clean(nameOrPath), nil
		}
		return "", ErrModelNotFound{Name: nameOrPath}
	}

	if strings.Contains(nameOrPath, string(os.PathSeparator)) || strings.Contains(nameOrPath, "/") {
		if info, err := os.Stat(nameOrPath); err == nil && info.IsDir() {
			abs, err := filepath.Abs(nameOrPath)
			if err != nil {
				return nameOrPath, nil
			}
			return abs, nil
		}
	}

	modelsDir, err := ModelsDir()
	if err != nil {
		return "", err
	}
	candidate := filepath.Join(modelsDir, nameOrPath)
	if info, err := os.Stat(candidate); err == nil && info.IsDir() {
		return candidate, nil
	}

	return "", ErrModelNotFound{Name: nameOrPath}
}

// ResolveModelOrDefault picks CLI model or config default.
func ResolveModelOrDefault(modelName string, cfg Config) (string, error) {
	name := modelName
	if name == "" {
		name = cfg.DefaultModel
	}
	if name == "" {
		models, err := ListModels()
		if err != nil {
			return "", err
		}
		if len(models) == 0 {
			return "", ErrNoModel{}
		}
		name = models[0]
	}
	return ResolveModel(name)
}

// FormatModelList formats model names for display.
func FormatModelList(models []string) string {
	if len(models) == 0 {
		return "  (none)"
	}
	var b strings.Builder
	for _, m := range models {
		b.WriteString("  - ")
		b.WriteString(m)
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}
