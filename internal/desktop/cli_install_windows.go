//go:build windows

package desktop

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func installCLIBinary(bundled, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("无法创建命令目录: %w", err)
	}
	in, err := os.Open(bundled)
	if err != nil {
		return fmt.Errorf("找不到随附的命令行程序: %w", err)
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return fmt.Errorf("无法替换已有的 akproxy 命令: %w", err)
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return fmt.Errorf("无法安装 akproxy 命令: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("无法安装 akproxy 命令: %w", closeErr)
	}
	return nil
}

func ensureUserPath(home string) error {
	return updateUserPath(filepath.Join(home, ".local", "bin"), true)
}

func removeUserPath(home string) error {
	return updateUserPath(filepath.Join(home, ".local", "bin"), false)
}

func updateUserPath(dir string, add bool) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("无法写入终端 PATH: %w", err)
	}
	defer key.Close()
	existing, _, err := key.GetStringValue("Path")
	if err != nil && err != registry.ErrNotExist {
		return fmt.Errorf("无法写入终端 PATH: %w", err)
	}
	var next string
	var changed bool
	if add {
		next, changed = MergePathDir(existing, dir)
	} else {
		next, changed = DropPathDir(existing, dir)
	}
	if !changed {
		return nil
	}
	if err := key.SetExpandStringValue("Path", next); err != nil {
		return fmt.Errorf("无法写入终端 PATH: %w", err)
	}
	notifyPathChange()
	return nil
}

func notifyPathChange() {
	env, err := windows.UTF16PtrFromString("Environment")
	if err != nil {
		return
	}
	user32 := windows.NewLazySystemDLL("user32.dll")
	send := user32.NewProc("SendMessageTimeoutW")
	const hwndBroadcast = 0xffff
	const wmSettingChange = 0x001a
	const smtoAbortIfHung = 0x0002
	_, _, _ = send.Call(uintptr(hwndBroadcast), uintptr(wmSettingChange), 0, uintptr(unsafe.Pointer(env)), uintptr(smtoAbortIfHung), 1000, 0)
}
