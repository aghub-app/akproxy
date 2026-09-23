package desktop

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeleteAccountConfinedToAuthDir(t *testing.T) {
	root := t.TempDir()
	authDir := filepath.Join(root, "auths")
	if err := os.MkdirAll(filepath.Join(authDir, "nested"), 0o700); err != nil {
		t.Fatal(err)
	}
	account := filepath.Join(authDir, "nested", "account.json")
	if err := os.WriteFile(account, []byte(`{"type":"codex"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "outside.json")
	if err := os.WriteFile(outside, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	r := &Runtime{paths: Paths{Auth: authDir}}
	for _, id := range []string{outside, "../outside.json", "missing.json"} {
		if err := r.DeleteAccount(id); err == nil {
			t.Fatalf("DeleteAccount(%q) unexpectedly succeeded", id)
		}
		if body, err := os.ReadFile(outside); err != nil || string(body) != "keep" {
			t.Fatalf("outside file changed after %q: %q, %v", id, body, err)
		}
	}
	if err := r.DeleteAccount(filepath.Join("nested", "account.json")); err != nil {
		t.Fatalf("delete listed nested account: %v", err)
	}
	if _, err := os.Stat(account); !os.IsNotExist(err) {
		t.Fatalf("account still exists: %v", err)
	}
}
