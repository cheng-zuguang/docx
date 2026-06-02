package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type genericIndex struct {
	SchemaVersion string      `json:"schemaVersion"`
	Items         []indexItem `json:"items"`
}

type indexItem struct {
	ID       string   `json:"id"`
	Title    string   `json:"title,omitempty"`
	Path     string   `json:"path"`
	Status   string   `json:"status,omitempty"`
	Category string   `json:"category,omitempty"`
	Modules  []string `json:"modules,omitempty"`
}

type mistakeIndex struct {
	SchemaVersion string      `json:"schemaVersion"`
	Categories    []string    `json:"categories"`
	Items         []indexItem `json:"items"`
}

func rebuildChangesIndex(root string, contextDir string) error {
	path, value, err := buildChangesIndex(root, contextDir)
	if err != nil {
		return err
	}
	return writeJSON(path, value)
}

func buildChangesIndex(root string, contextDir string) (string, genericIndex, error) {
	dir := filepath.Join(root, contextDir, "changes")
	records, err := collectJSONFiles[changeRecord](dir)
	if err != nil {
		return "", genericIndex{}, err
	}
	items := make([]indexItem, 0, len(records))
	for _, record := range records {
		items = append(items, indexItem{
			ID:      record.ID,
			Path:    contextDir + "/changes/" + record.ID + ".json",
			Modules: record.Modules,
		})
	}
	sortIndexItems(items)
	return filepath.Join(dir, "index.json"), genericIndex{SchemaVersion: schemaVersion, Items: items}, nil
}

func rebuildProposalsIndex(root string, contextDir string) error {
	path, value, err := buildProposalsIndex(root, contextDir)
	if err != nil {
		return err
	}
	return writeJSON(path, value)
}

func buildProposalsIndex(root string, contextDir string) (string, genericIndex, error) {
	dir := filepath.Join(root, contextDir, "proposals")
	records, err := collectJSONFiles[proposalRecord](dir)
	if err != nil {
		return "", genericIndex{}, err
	}
	items := make([]indexItem, 0, len(records))
	for _, record := range records {
		items = append(items, indexItem{
			ID:     record.ID,
			Path:   contextDir + "/proposals/" + record.ID + ".json",
			Status: record.Status,
		})
	}
	sortIndexItems(items)
	return filepath.Join(dir, "index.json"), genericIndex{SchemaVersion: schemaVersion, Items: items}, nil
}

func rebuildDecisionsIndex(root string, contextDir string) error {
	path, value, err := buildDecisionsIndex(root, contextDir)
	if err != nil {
		return err
	}
	return writeJSON(path, value)
}

func buildDecisionsIndex(root string, contextDir string) (string, genericIndex, error) {
	dir := filepath.Join(root, contextDir, "decisions")
	paths, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		return "", genericIndex{}, err
	}
	var items []indexItem
	for _, path := range paths {
		id := strings.TrimSuffix(filepath.Base(path), ".md")
		content, err := os.ReadFile(path)
		if err != nil {
			return "", genericIndex{}, err
		}
		items = append(items, indexItem{
			ID:     id,
			Title:  firstMarkdownHeading(string(content), id),
			Path:   contextDir + "/decisions/" + filepath.Base(path),
			Status: markdownStatus(string(content)),
		})
	}
	sortIndexItems(items)
	return filepath.Join(dir, "index.json"), genericIndex{SchemaVersion: schemaVersion, Items: items}, nil
}

func rebuildMistakesIndex(root string, contextDir string) error {
	path, value, err := buildMistakesIndex(root, contextDir)
	if err != nil {
		return err
	}
	return writeJSON(path, value)
}

func buildMistakesIndex(root string, contextDir string) (string, mistakeIndex, error) {
	dir := filepath.Join(root, contextDir, "mistakes")
	paths, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		return "", mistakeIndex{}, err
	}
	var items []indexItem
	categorySet := map[string]bool{}
	for _, path := range paths {
		category := strings.TrimSuffix(filepath.Base(path), ".md")
		categorySet[category] = true
		content, err := os.ReadFile(path)
		if err != nil {
			return "", mistakeIndex{}, err
		}
		for _, item := range parseMistakeItems(string(content), category, contextDir+"/mistakes/"+filepath.Base(path)) {
			items = append(items, item)
		}
	}
	sortIndexItems(items)
	return filepath.Join(dir, "index.json"), mistakeIndex{
		SchemaVersion: schemaVersion,
		Categories:    sortedKeys(categorySet),
		Items:         items,
	}, nil
}

func mustJSONString(value interface{}) string {
	var builder strings.Builder
	if err := writeJSONTo(&builder, value); err != nil {
		panic(err)
	}
	return builder.String()
}

func collectJSONFiles[T any](dir string) ([]T, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	var records []T
	for _, path := range paths {
		if filepath.Base(path) == "index.json" {
			continue
		}
		bytes, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var record T
		if err := json.Unmarshal(bytes, &record); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func sortIndexItems(items []indexItem) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
}

func firstMarkdownHeading(content string, fallback string) string {
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return fallback
}

func markdownStatus(content string) string {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(trimmed), "status:") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "Status:"))
		}
	}
	return ""
}

func parseMistakeItems(content string, category string, path string) []indexItem {
	var items []indexItem
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "## [") {
			continue
		}
		end := strings.Index(line, "]")
		if end < 4 {
			continue
		}
		id := line[4:end]
		title := strings.TrimSpace(line[end+1:])
		items = append(items, indexItem{ID: id, Title: title, Category: category, Path: path})
	}
	return items
}
