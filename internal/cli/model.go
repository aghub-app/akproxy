package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// ReadModelID reads cli.json. A missing file, broken JSON, or blank id all
// mean the user has not chosen a model.
func ReadModelID(path string) (string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", errNoModel
		}
		return "", fmt.Errorf("无法读取模型选择: %w", err)
	}
	var doc struct {
		Model struct {
			ID string `json:"id"`
		} `json:"model"`
	}
	if json.Unmarshal(body, &doc) != nil {
		return "", errNoModel
	}
	id := strings.TrimSpace(doc.Model.ID)
	if id == "" {
		return "", errNoModel
	}
	return id, nil
}

// WriteModelID stores the one selected model in cli.json.
func WriteModelID(path, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errNoModel
	}
	doc := struct {
		Model struct {
			ID string `json:"id"`
		} `json:"model"`
	}{}
	doc.Model.ID = id
	body, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("无法保存模型选择: %w", err)
	}
	if err := os.WriteFile(path, append(body, '\n'), 0o600); err != nil {
		return fmt.Errorf("无法保存模型选择: %w", err)
	}
	return nil
}
