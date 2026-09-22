package ci

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCommittedWorkflowPackagesDMG(t *testing.T) {
	root := moduleRoot(t)
	got, err := InspectMacOSDMG(root)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got.RunsOn, "macos-") {
		t.Fatalf("runs-on = %q", got.RunsOn)
	}
	if !strings.Contains(got.WailsBuild, "wails build") {
		t.Fatalf("build = %q", got.WailsBuild)
	}
	if !strings.HasSuffix(got.DMGPath, ".dmg") {
		t.Fatalf("dmg = %q", got.DMGPath)
	}
	if !strings.HasPrefix(got.UploadUses, "actions/upload-artifact@") {
		t.Fatalf("upload = %q", got.UploadUses)
	}
	if got.UploadPath != got.DMGPath {
		t.Fatalf("upload path %q is not the created dmg %q", got.UploadPath, got.DMGPath)
	}

	body, err := os.ReadFile(got.WorkflowFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{
		got.RunsOn,
		"wails build",
		"actions/upload-artifact",
		got.DMGPath,
		"go-version-file: go.mod",
		"pnpm/action-setup@",
		"actions/setup-node@",
		wailsModule,
		tagTrigger,
	} {
		if !bytes.Contains(body, []byte(needle)) {
			t.Fatalf("workflow file missing %q", needle)
		}
	}
	if bytes.Contains(body, []byte("workflow_dispatch")) || bytes.Contains(body, []byte("branches:")) {
		t.Fatal("packaging trigger is not limited to tags")
	}

	script, err := os.ReadFile(got.ScriptFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(script, []byte("hdiutil")) {
		t.Fatal("packaging script does not create an image with hdiutil")
	}
	if !bytes.Contains(body, []byte("scripts/package-dmg.sh")) {
		t.Fatal("workflow does not run scripts/package-dmg.sh")
	}
}

func TestInspectRejectsIncompleteWorkflows(t *testing.T) {
	script := "#!/bin/bash\nhdiutil create -volname akproxy -srcfolder \"$stage\" -ov -format UDZO \"$1\"\n"
	cases := []struct {
		name     string
		workflow string
		script   string
		want     string
	}{
		{
			name:     "linux runner",
			workflow: workflowYAML("ubuntu-latest", true, true, "build/bin/akproxy.dmg", "error"),
			script:   script,
			want:     "no macos-* runner",
		},
		{
			name:     "no wails build",
			workflow: workflowYAML("macos-latest", false, true, "build/bin/akproxy.dmg", "error"),
			script:   script,
			want:     "missing wails build",
		},
		{
			name:     "dmg only in a comment",
			workflow: workflowYAML("macos-latest", true, true, "build/bin/akproxy.dmg", "error"),
			script:   "#!/bin/bash\n# hdiutil create build/bin/akproxy.dmg\necho skip\n",
			want:     "missing hdiutil step that creates a .dmg",
		},
		{
			name:     "no upload",
			workflow: workflowYAML("macos-latest", true, false, "build/bin/akproxy.dmg", "error"),
			script:   script,
			want:     "missing actions/upload-artifact",
		},
		{
			name:     "uploads a different file",
			workflow: workflowYAML("macos-latest", true, true, "build/bin/other.dmg", "error"),
			script:   script,
			want:     "path is not the created .dmg",
		},
		{
			name:     "upload ignores a missing dmg",
			workflow: workflowYAML("macos-latest", true, true, "build/bin/akproxy.dmg", "warn"),
			script:   script,
			want:     "if-no-files-found must be error",
		},
		{
			name:     "branch push",
			workflow: strings.Replace(workflowYAML("macos-latest", true, true, "build/bin/akproxy.dmg", "error"), tagTrigger, "on: [push]\n", 1),
			script:   script,
			want:     "tag pushes",
		},
		{
			name:     "manual dispatch",
			workflow: strings.Replace(workflowYAML("macos-latest", true, true, "build/bin/akproxy.dmg", "error"), "      - \"**\"\n", "      - \"**\"\n  workflow_dispatch:\n", 1),
			script:   script,
			want:     "tag pushes",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := writeFixture(t, tc.workflow, tc.script)
			_, err := InspectMacOSDMG(root)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not contain %q", err, tc.want)
			}
		})
	}
}

func TestInspectAcceptsFixtureThatCreatesAndUploadsDMG(t *testing.T) {
	root := writeFixture(t, workflowYAML("macos-14", true, true, "build/bin/akproxy.dmg", "error"), "#!/bin/bash\nhdiutil create -format UDZO out.dmg\n")
	got, err := InspectMacOSDMG(root)
	if err != nil {
		t.Fatal(err)
	}
	if got.RunsOn != "macos-14" || got.DMGPath != "build/bin/akproxy.dmg" || got.UploadPath != got.DMGPath {
		t.Fatalf("%+v", got)
	}
	if !strings.Contains(got.WailsBuild, "wails build") {
		t.Fatalf("build = %q", got.WailsBuild)
	}
	if !strings.HasPrefix(got.UploadUses, "actions/upload-artifact@") {
		t.Fatalf("upload = %q", got.UploadUses)
	}
}

const tagTrigger = "on:\n  push:\n    tags:\n      - \"**\"\n"

func workflowYAML(runsOn string, withBuild, withUpload bool, uploadPath, ifNoFiles string) string {
	var b strings.Builder
	b.WriteString("name: package\n")
	b.WriteString(tagTrigger)
	b.WriteString("jobs:\n  package:\n    runs-on: " + runsOn + "\n    steps:\n")
	b.WriteString("      - uses: actions/checkout@v7.0.1\n")
	b.WriteString("      - uses: pnpm/action-setup@v6.1.0\n")
	b.WriteString("      - uses: actions/setup-node@v7.0.0\n")
	b.WriteString("      - uses: actions/setup-go@v7.0.0\n        with:\n          go-version-file: go.mod\n")
	b.WriteString("      - run: go install " + wailsModule + "\n")
	if withBuild {
		b.WriteString("      - run: wails build\n")
	}
	b.WriteString("      - run: bash scripts/package-dmg.sh build/bin/akproxy.dmg\n")
	b.WriteString("      # hdiutil create ignored.dmg\n")
	if withUpload {
		b.WriteString("      - uses: actions/upload-artifact@v7.0.1\n        with:\n          path: " + uploadPath + "\n          if-no-files-found: " + ifNoFiles + "\n")
	}
	return b.String()
}

func writeFixture(t *testing.T, workflow, script string) string {
	t.Helper()
	root := t.TempDir()
	workflowDir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workflowDir, "macos-dmg.yml"), []byte(workflow), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scripts", "package-dmg.sh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
