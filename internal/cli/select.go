package cli

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"slices"

	"akproxy/internal/desktop"
)

type modelPicker func(models []string, current string) (string, error)

func selectModel(stdout io.Writer, paths desktop.Paths, pick modelPicker) error {
	if err := requireApp(paths.Config); err != nil {
		return err
	}
	address, key, err := desktop.ClientCredential(paths.Config)
	if err != nil {
		return err
	}
	models, err := desktop.FetchModels(address, key)
	if err != nil {
		return err
	}
	if len(models) == 0 {
		return errors.New("暂无可选模型")
	}
	current, err := ReadModelID(filepath.Join(paths.Root, "cli.json"))
	if err != nil && !errors.Is(err, errNoModel) {
		return err
	}
	if !slices.Contains(models, current) {
		current = ""
	}
	chosen, err := pick(models, current)
	if err != nil {
		return err
	}
	if !slices.Contains(models, chosen) {
		return errors.New("暂无可选模型")
	}
	path := filepath.Join(paths.Root, "cli.json")
	if err := WriteModelID(path, chosen); err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, chosen)
	return err
}
