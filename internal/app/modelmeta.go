package app

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const defaultContextLimit = 32768

// ContextLimitFromModel reads max_position_embeddings from a model's config.json.
func ContextLimitFromModel(modelPath string) int {
	configPath := filepath.Join(modelPath, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return defaultContextLimit
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return defaultContextLimit
	}

	if v := intFromJSON(raw["max_position_embeddings"]); v > 0 {
		return v
	}

	if textRaw, ok := raw["text_config"]; ok {
		var textCfg map[string]json.RawMessage
		if err := json.Unmarshal(textRaw, &textCfg); err == nil {
			if v := intFromJSON(textCfg["max_position_embeddings"]); v > 0 {
				return v
			}
		}
	}

	return defaultContextLimit
}

func intFromJSON(raw json.RawMessage) int {
	if len(raw) == 0 {
		return 0
	}
	var n int
	if err := json.Unmarshal(raw, &n); err == nil && n > 0 {
		return n
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil && f > 0 {
		return int(f)
	}
	return 0
}
