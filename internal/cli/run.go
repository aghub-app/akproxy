package cli

import (
	"errors"
	"fmt"
	"os"
)

var errNoModel = errors.New("还没有选择模型，请先运行 akproxy model")

func requireApp(configPath string) error {
	if _, err := os.Stat(configPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return errors.New("请先打开 akproxy")
		}
		return fmt.Errorf("读取配置失败: %w", err)
	}
	return nil
}
