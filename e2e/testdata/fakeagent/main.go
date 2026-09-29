// fakeagent stands in for claude, codex, opencode, and pi. It records how it was
// launched to $AKPROXY_FAKE_OUT and exits with $AKPROXY_FAKE_EXIT.
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type record struct {
	Name  string            `json:"name"`
	Args  []string          `json:"args"`
	Env   map[string]string `json:"env"`
	Files map[string]string `json:"files"`
}

func main() {
	name := strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe")
	rec := record{Name: name, Args: os.Args[1:], Env: map[string]string{}, Files: map[string]string{}}
	for _, item := range os.Environ() {
		key, value, _ := strings.Cut(item, "=")
		rec.Env[key] = value
	}
	if dir := os.Getenv("PI_CODING_AGENT_DIR"); dir != "" {
		for _, file := range []string{"models.json", "settings.json"} {
			body, _ := os.ReadFile(filepath.Join(dir, file))
			rec.Files[file] = string(body)
		}
	}
	if out := os.Getenv("AKPROXY_FAKE_OUT"); out != "" {
		body, _ := json.Marshal(rec)
		_ = os.WriteFile(out, body, 0o644)
	}
	code, _ := strconv.Atoi(os.Getenv("AKPROXY_FAKE_EXIT"))
	os.Exit(code)
}
