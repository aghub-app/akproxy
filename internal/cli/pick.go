package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/gum/v2/choose"
	"charm.land/gum/v2/style"
)

func gumStyle(foreground string) style.Styles {
	return style.Styles{
		Foreground: foreground,
		Border:     "none",
		Align:      "left",
		Margin:     "0 0",
		Padding:    "0 0",
	}
}

// pickWithGum opens gum's choose list with its default colors.
func pickWithGum(models []string, current string) (string, error) {
	options := choose.Options{
		Options:           append([]string(nil), models...),
		Limit:             1,
		Height:            10,
		Cursor:            "> ",
		ShowHelp:          true,
		Header:            "Choose:",
		InputDelimiter:    "\n",
		OutputDelimiter:   "\n",
		Padding:           "0 0",
		CursorStyle:       gumStyle("212"),
		HeaderStyle:       gumStyle("99"),
		ItemStyle:         gumStyle(""),
		SelectedItemStyle: gumStyle("212"),
	}
	if current != "" {
		options.Selected = []string{current}
	}
	reader, writer, err := os.Pipe()
	if err != nil {
		return "", fmt.Errorf("无法打开模型选择: %w", err)
	}
	previous := os.Stdout
	os.Stdout = writer
	runErr := options.Run()
	os.Stdout = previous
	_ = writer.Close()
	body, readErr := io.ReadAll(reader)
	_ = reader.Close()
	if runErr != nil {
		if errors.Is(runErr, tea.ErrInterrupted) || strings.Contains(runErr.Error(), "nothing selected") {
			return "", errors.New("已取消")
		}
		return "", fmt.Errorf("无法打开模型选择: %w", runErr)
	}
	if readErr != nil {
		return "", fmt.Errorf("无法打开模型选择: %w", readErr)
	}
	chosen := strings.TrimSpace(string(body))
	if chosen == "" {
		return "", errors.New("已取消")
	}
	return chosen, nil
}
