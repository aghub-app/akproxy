package ci

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const wailsModule = "github.com/wailsapp/wails/v2/cmd/wails@v2.12.0"

// Report is the macOS job that builds this repo with wails and uploads a DMG.
type Report struct {
	WorkflowFile string
	JobName      string
	RunsOn       string
	WailsBuild   string
	DMGPath      string
	ScriptFile   string
	UploadUses   string
	UploadPath   string
}

type workflowFile struct {
	Jobs map[string]workflowJob `yaml:"jobs"`
}

type workflowJob struct {
	RunsOn any            `yaml:"runs-on"`
	Steps  []workflowStep `yaml:"steps"`
}

type workflowStep struct {
	Uses string         `yaml:"uses"`
	Run  string         `yaml:"run"`
	With map[string]any `yaml:"with"`
}

// InspectMacOSDMG reads the committed workflows and returns the job that
// builds on a macos-* runner, creates a DMG with hdiutil, and uploads that
// DMG with actions/upload-artifact.
func InspectMacOSDMG(repoRoot string) (Report, error) {
	matches, err := workflowPaths(repoRoot)
	if err != nil {
		return Report{}, err
	}
	if len(matches) == 0 {
		return Report{}, fmt.Errorf("no workflows under .github/workflows")
	}

	var macosErrs []string
	for _, file := range matches {
		report, problems, macos, err := inspectFile(repoRoot, file)
		if err != nil {
			return Report{}, err
		}
		if macos && len(problems) == 0 {
			return report, nil
		}
		if macos {
			macosErrs = append(macosErrs, fmt.Sprintf("%s: %s", filepath.Base(file), strings.Join(problems, "; ")))
		}
	}
	if len(macosErrs) == 0 {
		return Report{}, fmt.Errorf("no macos-* runner")
	}
	return Report{}, fmt.Errorf("%s", strings.Join(macosErrs, "; "))
}

func workflowPaths(repoRoot string) ([]string, error) {
	var matches []string
	for _, pattern := range []string{"*.yml", "*.yaml"} {
		found, err := filepath.Glob(filepath.Join(repoRoot, ".github", "workflows", pattern))
		if err != nil {
			return nil, err
		}
		matches = append(matches, found...)
	}
	return matches, nil
}

func inspectFile(repoRoot, file string) (Report, []string, bool, error) {
	body, err := os.ReadFile(file)
	if err != nil {
		return Report{}, nil, false, err
	}
	var doc workflowFile
	if err := yaml.Unmarshal(body, &doc); err != nil {
		return Report{}, nil, false, fmt.Errorf("%s: %w", filepath.Base(file), err)
	}

	var macosErrs []string
	sawMacOS := false
	for name, job := range doc.Jobs {
		runners := runsOnValues(job.RunsOn)
		macos := false
		var runsOn string
		for _, runner := range runners {
			if strings.HasPrefix(runner, "macos-") {
				macos = true
				runsOn = runner
				break
			}
		}
		if !macos {
			continue
		}
		sawMacOS = true
		report, problems, err := inspectJob(repoRoot, file, name, runsOn, job)
		if err != nil {
			return Report{}, nil, true, err
		}
		if len(problems) == 0 {
			return report, nil, true, nil
		}
		macosErrs = append(macosErrs, fmt.Sprintf("job %s: %s", name, strings.Join(problems, "; ")))
	}
	return Report{}, macosErrs, sawMacOS, nil
}

func inspectJob(repoRoot, file, name, runsOn string, job workflowJob) (Report, []string, error) {
	var problems []string
	report := Report{WorkflowFile: file, JobName: name, RunsOn: runsOn}

	if !hasAction(job, "actions/checkout") {
		problems = append(problems, "missing actions/checkout")
	}
	if !hasGoFromMod(job) {
		problems = append(problems, "missing actions/setup-go with go-version-file go.mod")
	}
	if !hasAction(job, "actions/setup-node") {
		problems = append(problems, "missing actions/setup-node")
	}
	if !hasAction(job, "pnpm/action-setup") {
		problems = append(problems, "missing pnpm/action-setup")
	}
	if !hasCommand(job, wailsModule) {
		problems = append(problems, "missing go install "+wailsModule)
	}

	build, ok := commandText(job, "wails build")
	if !ok {
		problems = append(problems, "missing wails build")
	} else {
		report.WailsBuild = build
	}

	dmgs, script, err := createdDMGs(repoRoot, job)
	if err != nil {
		return Report{}, nil, err
	}
	report.ScriptFile = script
	if len(dmgs) == 0 {
		problems = append(problems, "missing hdiutil step that creates a .dmg")
	}

	uses, uploadPath, ifNoFiles, found := uploadArtifact(job)
	if !found {
		problems = append(problems, "missing actions/upload-artifact")
	} else {
		report.UploadUses = uses
		report.UploadPath = uploadPath
		switch {
		case !strings.HasSuffix(uploadPath, ".dmg"):
			problems = append(problems, "actions/upload-artifact path is not a .dmg")
		case !contains(dmgs, uploadPath):
			problems = append(problems, "actions/upload-artifact path is not the created .dmg")
		case ifNoFiles != "error":
			problems = append(problems, "actions/upload-artifact if-no-files-found must be error")
		default:
			report.DMGPath = uploadPath
		}
	}
	return report, problems, nil
}

func runsOnValues(value any) []string {
	switch typed := value.(type) {
	case string:
		return []string{typed}
	case []any:
		var out []string
		for _, item := range typed {
			if text, ok := item.(string); ok {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}

func hasAction(job workflowJob, action string) bool {
	for _, step := range job.Steps {
		if usesAction(step.Uses, action) {
			return true
		}
	}
	return false
}

func hasGoFromMod(job workflowJob) bool {
	for _, step := range job.Steps {
		if !usesAction(step.Uses, "actions/setup-go") {
			continue
		}
		file, ok := yamlString(step.With["go-version-file"])
		if ok && (file == "go.mod" || strings.HasSuffix(file, "/go.mod")) {
			return true
		}
	}
	return false
}

func hasCommand(job workflowJob, needle string) bool {
	_, ok := commandText(job, needle)
	return ok
}

func commandText(job workflowJob, needle string) (string, bool) {
	for _, step := range job.Steps {
		for _, line := range strings.Split(step.Run, "\n") {
			line = stripShellComment(line)
			if strings.Contains(line, needle) {
				return strings.TrimSpace(line), true
			}
		}
	}
	return "", false
}

func stripShellComment(line string) string {
	if idx := strings.Index(line, "#"); idx >= 0 {
		return line[:idx]
	}
	return line
}

func shellCode(text string) string {
	var b strings.Builder
	for _, line := range strings.Split(text, "\n") {
		b.WriteString(stripShellComment(line))
		b.WriteByte('\n')
	}
	return b.String()
}

func createdDMGs(repoRoot string, job workflowJob) ([]string, string, error) {
	var dmgs []string
	var script string
	for _, step := range job.Steps {
		run := shellCode(step.Run)
		if strings.Contains(run, "hdiutil") {
			dmgs = append(dmgs, dmgPaths(run)...)
		}
		for _, rel := range scriptPaths(run) {
			full := filepath.Join(repoRoot, filepath.FromSlash(rel))
			body, err := os.ReadFile(full)
			if err != nil {
				return nil, "", fmt.Errorf("read %s: %w", rel, err)
			}
			code := shellCode(string(body))
			if !strings.Contains(code, "hdiutil") {
				continue
			}
			script = full
			args := dmgPaths(run)
			if len(args) == 0 {
				args = dmgPaths(code)
			}
			dmgs = append(dmgs, args...)
		}
	}
	return unique(dmgs), script, nil
}

func uploadArtifact(job workflowJob) (uses, uploadPath, ifNoFiles string, found bool) {
	for _, step := range job.Steps {
		if !usesAction(step.Uses, "actions/upload-artifact") {
			continue
		}
		pathValue, _ := yamlString(step.With["path"])
		ifNoFiles, _ = yamlString(step.With["if-no-files-found"])
		return strings.TrimSpace(step.Uses), cleanRel(pathValue), ifNoFiles, true
	}
	return "", "", "", false
}

func usesAction(uses, action string) bool {
	uses = strings.TrimSpace(uses)
	return uses == action || strings.HasPrefix(uses, action+"@") || strings.HasPrefix(uses, action+"/")
}

func scriptPaths(run string) []string {
	var out []string
	for _, field := range strings.Fields(run) {
		field = strings.Trim(field, `"'`)
		if !strings.Contains(field, ".sh") {
			continue
		}
		field = strings.TrimPrefix(field, "./")
		if strings.HasPrefix(field, "/") || strings.Contains(field, "..") {
			continue
		}
		out = append(out, path.Clean(field))
	}
	return out
}

func dmgPaths(text string) []string {
	var out []string
	for _, field := range strings.Fields(text) {
		field = strings.Trim(field, `"'`)
		field = strings.TrimRight(field, `;|&`)
		if !strings.HasSuffix(field, ".dmg") || strings.Contains(field, "$") {
			continue
		}
		field = strings.TrimPrefix(field, "./")
		if strings.HasPrefix(field, "/") || strings.Contains(field, "..") {
			continue
		}
		out = append(out, path.Clean(field))
	}
	return out
}

func cleanRel(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"'`)
	value = strings.TrimPrefix(value, "./")
	if value == "" {
		return ""
	}
	return path.Clean(value)
}

func yamlString(value any) (string, bool) {
	text, ok := value.(string)
	return text, ok
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func unique(values []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
