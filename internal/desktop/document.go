package desktop

import (
	"fmt"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/config"
)

func loadConfig(path string) (*config.Config, error) {
	cfg, err := config.LoadConfig(path)
	if err != nil {
		return nil, fmt.Errorf("配置无法读取: %w", err)
	}
	return cfg, nil
}

func saveConfig(path string, cfg *config.Config) error {
	if err := config.SaveConfigPreserveComments(path, cfg); err != nil {
		return fmt.Errorf("配置无法保存: %w", err)
	}
	return nil
}
