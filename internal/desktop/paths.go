package desktop

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

const appDirName = "akproxy"

// Paths are the app-owned config and account locations.
type Paths struct {
	Root   string
	Config string
	Auth   string
}

// Resolve returns the user-config locations for this app.
func Resolve() (Paths, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return Paths{}, fmt.Errorf("找不到用户配置目录: %w", err)
	}
	root := filepath.Join(base, appDirName)
	return Paths{
		Root:   root,
		Config: filepath.Join(root, "config.yaml"),
		Auth:   filepath.Join(root, "auths"),
	}, nil
}

// Ensure creates the data directories and the first config file.
func Ensure(p Paths) error {
	if err := os.MkdirAll(p.Auth, 0o700); err != nil {
		return fmt.Errorf("创建账号目录失败: %w", err)
	}
	if _, err := os.Stat(p.Config); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("读取配置失败: %w", err)
	}
	key, err := newClientKey()
	if err != nil {
		return err
	}
	body := defaultConfig(p.Auth, key)
	if err := os.WriteFile(p.Config, []byte(body), 0o600); err != nil {
		return fmt.Errorf("写入初始配置失败: %w", err)
	}
	return nil
}

func newClientKey() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("生成客户端密钥失败: %w", err)
	}
	return "sk-" + hex.EncodeToString(buf), nil
}

func defaultConfig(authDir, clientKey string) string {
	return fmt.Sprintf(`# akproxy 自己的配置。账号目录由应用固定，不要改到别处。
host: "127.0.0.1"
port: 8317
auth-dir: %q
api-keys:
  - %q
debug: false
proxy-url: ""
request-retry: 3
routing:
  strategy: "round-robin"
remote-management:
  allow-remote: false
  secret-key: ""
  disable-control-panel: true
ws-auth: true
`, authDir, clientKey)
}
