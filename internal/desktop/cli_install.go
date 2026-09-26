package desktop

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	cliPathBegin = "# akproxy cli"
	cliPathEnd   = "# end akproxy cli"
	cliMarker    = ".akproxy-cli"
)

var cliInstall struct {
	mu    sync.Mutex
	error string
}

// CLIStatus is the command-line switch and the latest install failure.
type CLIStatus struct {
	Enabled bool   `json:"enabled"`
	Error   string `json:"error"`
}

// CurrentCLIStatus returns the last install or removal error for this process.
func CurrentCLIStatus(enabled bool) CLIStatus {
	cliInstall.mu.Lock()
	defer cliInstall.mu.Unlock()
	return CLIStatus{Enabled: enabled, Error: cliInstall.error}
}

// RememberCLIError keeps a startup or switch failure for the settings tab.
func RememberCLIError(err error) {
	cliInstall.mu.Lock()
	defer cliInstall.mu.Unlock()
	if err == nil {
		cliInstall.error = ""
		return
	}
	cliInstall.error = err.Error()
}

// ApplyCLIInstall puts the bundled CLI on the user PATH, or removes it.
// version is the running app version. "dev" always replaces the installed command.
// Any other version replaces it only when `akproxy --version` does not match.
func ApplyCLIInstall(enabled bool, version string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("找不到用户目录: %w", err)
	}
	bundled := ""
	if enabled {
		bundled, err = BundledCLI()
		if err != nil {
			return err
		}
	}
	return syncCLIInstall(home, bundled, enabled, version)
}

// BundledCLI is the CLI shipped beside the desktop executable.
func BundledCLI() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("找不到随附的命令行程序: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	name := "akproxy-cli"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	candidate := filepath.Join(filepath.Dir(exe), name)
	info, err := os.Stat(candidate)
	if err != nil || info.IsDir() {
		return "", fmt.Errorf("找不到随附的命令行程序")
	}
	return candidate, nil
}

func cliCommandPath(home string) string {
	name := "akproxy"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(home, ".local", "bin", name)
}

func cliMarkerPath(home string) string {
	return filepath.Join(home, ".local", "bin", cliMarker)
}

func installedCLIVersion(path string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, "--version").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func syncCLIInstall(home, bundled string, enabled bool, version string) error {
	if enabled {
		dest := cliCommandPath(home)
		if version != "dev" && installedCLIVersion(dest) == version {
			return ensureUserPath(home)
		}
		if err := installCLIBinary(bundled, dest); err != nil {
			return err
		}
		if err := os.WriteFile(cliMarkerPath(home), []byte(dest+"\n"), 0o644); err != nil {
			return fmt.Errorf("无法安装 akproxy 命令: %w", err)
		}
		if err := ensureUserPath(home); err != nil {
			return err
		}
		return nil
	}
	if err := removeInstalledCLI(home); err != nil {
		return err
	}
	return removeUserPath(home)
}

func removeInstalledCLI(home string) error {
	marker := cliMarkerPath(home)
	body, err := os.ReadFile(marker)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("无法移除 akproxy 命令: %w", err)
	}
	dest := strings.TrimSpace(string(body))
	if dest != cliCommandPath(home) {
		return fmt.Errorf("无法移除 akproxy 命令")
	}
	if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("无法移除 akproxy 命令: %w", err)
	}
	if err := os.Remove(marker); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("无法移除 akproxy 命令: %w", err)
	}
	return nil
}

func pathBlock() string {
	return cliPathBegin + "\nexport PATH=\"$HOME/.local/bin:$PATH\"\n" + cliPathEnd + "\n"
}

func stripCLIBlock(body string) string {
	for {
		start := strings.Index(body, cliPathBegin)
		if start < 0 {
			return body
		}
		rel := strings.Index(body[start:], cliPathEnd)
		if rel < 0 {
			return body
		}
		end := start + rel + len(cliPathEnd)
		if end < len(body) && body[end] == '\n' {
			end++
		}
		body = body[:start] + body[end:]
	}
}

func shellStartupFiles(home string) []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{filepath.Join(home, ".zprofile"), filepath.Join(home, ".zshrc")}
	case "linux":
		return []string{filepath.Join(home, ".profile"), filepath.Join(home, ".bashrc")}
	default:
		return nil
	}
}

func ensureShellPath(home string) error {
	block := pathBlock()
	for _, path := range shellStartupFiles(home) {
		body, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("无法写入终端 PATH: %w", err)
		}
		text := string(body)
		if strings.Contains(text, cliPathBegin) {
			continue
		}
		next := text
		if next != "" && !strings.HasSuffix(next, "\n") {
			next += "\n"
		}
		if err := os.WriteFile(path, []byte(next+block), 0o644); err != nil {
			return fmt.Errorf("无法写入终端 PATH: %w", err)
		}
	}
	return nil
}

func removeShellPath(home string) error {
	for _, path := range shellStartupFiles(home) {
		body, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("无法写入终端 PATH: %w", err)
		}
		next := stripCLIBlock(string(body))
		if next == string(body) {
			continue
		}
		if strings.TrimSpace(next) == "" {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("无法写入终端 PATH: %w", err)
			}
			continue
		}
		if err := os.WriteFile(path, []byte(next), 0o644); err != nil {
			return fmt.Errorf("无法写入终端 PATH: %w", err)
		}
	}
	return nil
}

// MergePathDir prepends dir when it is not already listed.
func MergePathDir(existing, dir string) (string, bool) {
	if pathHasDir(existing, dir) {
		return existing, false
	}
	if strings.TrimSpace(existing) == "" {
		return dir, true
	}
	return dir + string(os.PathListSeparator) + existing, true
}

// DropPathDir removes dir from a PATH list.
func DropPathDir(existing, dir string) (string, bool) {
	parts := strings.Split(existing, string(os.PathListSeparator))
	kept := make([]string, 0, len(parts))
	changed := false
	for _, part := range parts {
		if pathEqual(part, dir) {
			changed = true
			continue
		}
		if part == "" {
			continue
		}
		kept = append(kept, part)
	}
	if !changed {
		return existing, false
	}
	return strings.Join(kept, string(os.PathListSeparator)), true
}

func pathHasDir(existing, dir string) bool {
	for _, part := range strings.Split(existing, string(os.PathListSeparator)) {
		if pathEqual(part, dir) {
			return true
		}
	}
	return false
}

func pathEqual(left, right string) bool {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
	}
	return filepath.Clean(left) == filepath.Clean(right)
}
