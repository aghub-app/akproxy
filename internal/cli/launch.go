package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// CodexKeyEnv is the environment variable Codex reads for this launch.
// The key stays out of the process arguments.
const CodexKeyEnv = "AKPROXY_CLI_KEY"

// Launch is the process that would be started for one CLI.
type Launch struct {
	Argv  []string
	Env   map[string]string
	Clear []string
	Dir   string
}

// BuildLaunch assembles the Claude or Codex process for one selected model.
// baseURL is the proxy origin, without a path.
func BuildLaunch(name, baseURL, apiKey, model string, extra []string) (Launch, error) {
	if strings.TrimSpace(baseURL) == "" || strings.TrimSpace(apiKey) == "" || strings.TrimSpace(model) == "" {
		return Launch{}, fmt.Errorf("启动环境不完整")
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	extra = append([]string(nil), extra...)
	switch name {
	case "claude":
		return Launch{
			Argv: append([]string{"claude", "--model", model}, extra...),
			Env: map[string]string{
				"ANTHROPIC_BASE_URL":             baseURL,
				"ANTHROPIC_AUTH_TOKEN":           apiKey,
				"ANTHROPIC_MODEL":                model,
				"ANTHROPIC_DEFAULT_OPUS_MODEL":   model,
				"ANTHROPIC_DEFAULT_SONNET_MODEL": model,
				"ANTHROPIC_DEFAULT_HAIKU_MODEL":  model,
				"CLAUDE_CODE_SUBAGENT_MODEL":     model,
			},
			Clear: []string{"ANTHROPIC_API_KEY"},
		}, nil
	case "codex":
		argv := []string{
			"codex",
			"-c", "model_provider=akproxy",
			"-c", configValue("model_providers.akproxy.name", "akproxy"),
			"-c", configValue("model_providers.akproxy.base_url", baseURL+"/v1"),
			"-c", configValue("model_providers.akproxy.env_key", CodexKeyEnv),
			"-c", configValue("model_providers.akproxy.wire_api", "responses"),
			"-c", configValue("model", model),
		}
		return Launch{
			Argv: append(argv, extra...),
			Env:  map[string]string{CodexKeyEnv: apiKey},
		}, nil
	case "opencode":
		content, err := openCodeConfig(baseURL+"/v1", apiKey, model)
		if err != nil {
			return Launch{}, err
		}
		return Launch{
			Argv: append([]string{"opencode"}, extra...),
			Env:  map[string]string{"OPENCODE_CONFIG_CONTENT": content},
		}, nil
	case "pi":
		dir, err := writePiConfig(baseURL+"/v1", model)
		if err != nil {
			return Launch{}, err
		}
		return Launch{
			Argv: append([]string{"pi"}, extra...),
			Env: map[string]string{
				CodexKeyEnv:           apiKey,
				"PI_CODING_AGENT_DIR": dir,
			},
			Dir: dir,
		}, nil
	default:
		return Launch{}, fmt.Errorf("未知命令 %q", name)
	}
}

func openCodeConfig(baseURL, apiKey, model string) (string, error) {
	config := map[string]any{
		"$schema": "https://opencode.ai/config.json",
		"provider": map[string]any{
			"akproxy": map[string]any{
				"npm":  "@ai-sdk/openai-compatible",
				"name": "akproxy",
				"options": map[string]any{
					"baseURL": baseURL,
					"apiKey":  apiKey,
				},
				"models": map[string]any{
					model: map[string]any{"name": model},
				},
			},
		},
		"model": "akproxy/" + model,
	}
	body, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("无法准备 OpenCode 配置: %w", err)
	}
	return string(body), nil
}

func writePiConfig(baseURL, model string) (string, error) {
	dir, err := os.MkdirTemp("", "akproxy-pi-")
	if err != nil {
		return "", fmt.Errorf("无法准备 Pi 配置: %w", err)
	}
	models := map[string]any{
		"providers": map[string]any{
			"akproxy": map[string]any{
				"baseUrl": baseURL,
				"api":     "openai-completions",
				"apiKey":  "$" + CodexKeyEnv,
				"models":  []any{map[string]any{"id": model}},
			},
		},
	}
	settings := map[string]any{
		"defaultProvider": "akproxy",
		"defaultModel":    model,
	}
	if err := writeJSON(filepath.Join(dir, "models.json"), models); err != nil {
		os.RemoveAll(dir)
		return "", err
	}
	if err := writeJSON(filepath.Join(dir, "settings.json"), settings); err != nil {
		os.RemoveAll(dir)
		return "", err
	}
	return dir, nil
}

func writeJSON(path string, value any) error {
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("无法准备 Pi 配置: %w", err)
	}
	if err := os.WriteFile(path, append(body, '\n'), 0o600); err != nil {
		return fmt.Errorf("无法准备 Pi 配置: %w", err)
	}
	return nil
}

func configValue(key, value string) string {
	if strings.ContainsAny(value, " \t\"'") {
		value = strconv.Quote(value)
	}
	return key + "=" + value
}
