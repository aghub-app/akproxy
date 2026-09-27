//go:build windows

package desktop

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func replaceCLIBinary(tmp, dest string) error {
	if err := moveAsideCLI(dest); err != nil {
		return err
	}
	return os.Rename(tmp, dest)
}

// Windows cannot overwrite or delete an executable that a terminal is still
// running, but it can rename it. The old copy is removed on a later install.
func moveAsideCLI(dest string) error {
	clearOldCLI(dest)
	if _, err := os.Lstat(dest); os.IsNotExist(err) {
		return nil
	}
	return os.Rename(dest, fmt.Sprintf("%s.old-%d", dest, time.Now().UnixNano()))
}

func clearOldCLI(dest string) {
	old, _ := filepath.Glob(dest + ".old-*")
	for _, path := range old {
		_ = os.Remove(path)
	}
}

func removeCLIBinary(dest string) error {
	if err := moveAsideCLI(dest); err != nil {
		return err
	}
	clearOldCLI(dest)
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
