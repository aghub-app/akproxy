//go:build !windows

package desktop

import (
	"fmt"
	"os"
	"path/filepath"
)

func installCLIBinary(bundled, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("无法创建命令目录: %w", err)
	}
	if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("无法替换已有的 akproxy 命令: %w", err)
	}
	if err := os.Symlink(bundled, dest); err != nil {
		return fmt.Errorf("无法安装 akproxy 命令: %w", err)
	}
	return nil
}

func ensureUserPath(home string) error {
	return ensureShellPath(home)
}

func removeUserPath(home string) error {
	return removeShellPath(home)
}
