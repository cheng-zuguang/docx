package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func loadConfig(root string) (configFile, error) {
	var config configFile
	bytes, err := os.ReadFile(filepath.Join(root, ".docx.json"))
	if err != nil {
		return configFile{}, err
	}
	if err := json.Unmarshal(bytes, &config); err != nil {
		return configFile{}, err
	}
	if config.ContextDir == "" {
		config.ContextDir = defaultDocDir
	}
	return config, nil
}

func readIndex(path string) (indexFile, error) {
	var index indexFile
	bytes, err := os.ReadFile(path)
	if err != nil {
		return indexFile{}, err
	}
	if err := json.Unmarshal(bytes, &index); err != nil {
		return indexFile{}, err
	}
	return index, nil
}
