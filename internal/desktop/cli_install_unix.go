//go:build !windows

package desktop

import "os"

func replaceCLIBinary(tmp, dest string) error {
	return os.Rename(tmp, dest)
}

func removeCLIBinary(dest string) error {
	if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func ensureUserPath(home string) error {
	return ensureShellPath(home)
}

func removeUserPath(home string) error {
	return removeShellPath(home)
}
