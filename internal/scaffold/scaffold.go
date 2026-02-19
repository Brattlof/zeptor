package scaffold

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

type Options struct {
	ProjectName string
	Template    string
	Port        int
	SkipGit     bool
	SkipTempl   bool
	OutputDir   string
}

type templateData struct {
	ProjectName string
	Port        int
}

var projectNameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-_]*$`)

func ValidateProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("project name cannot be empty")
	}
	if len(name) > 100 {
		return fmt.Errorf("project name too long (max 100 characters)")
	}
	if !projectNameRegex.MatchString(name) {
		return fmt.Errorf("project name must be lowercase alphanumeric, dashes, or underscores (e.g., my-project)")
	}
	return nil
}

func ValidateProjectNameForModule(name string) string {
	return strings.ReplaceAll(name, "-", "")
}

func Create(opts Options) error {
	if err := ValidateProjectName(opts.ProjectName); err != nil {
		return err
	}

	if !TemplateExists(opts.Template) {
		return fmt.Errorf("unknown template: %s (available: %v)", opts.Template, AvailableTemplates())
	}

	targetDir := opts.OutputDir
	if targetDir == "" {
		targetDir = opts.ProjectName
	}

	if _, err := os.Stat(targetDir); err == nil {
		return fmt.Errorf("directory %s already exists", targetDir)
	}

	data := templateData{
		ProjectName: opts.ProjectName,
		Port:        opts.Port,
	}

	templateFS, err := GetTemplates(opts.Template)
	if err != nil {
		return fmt.Errorf("failed to load template: %w", err)
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := copyTemplateDir(templateFS, ".", targetDir, data); err != nil {
		cleanupOnError(targetDir)
		return fmt.Errorf("failed to copy template: %w", err)
	}

	if err := createGoMod(opts.ProjectName, targetDir); err != nil {
		cleanupOnError(targetDir)
		return fmt.Errorf("failed to create go.mod: %w", err)
	}

	if !opts.SkipTempl {
		if err := runTemplGenerate(targetDir); err != nil {
			cleanupOnError(targetDir)
			return fmt.Errorf("failed to run templ generate: %w", err)
		}
	}

	if !opts.SkipGit {
		if err := initGit(targetDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to initialize git: %v\n", err)
		}
	}

	return nil
}

func copyTemplateDir(srcFS fs.FS, srcPath, destPath string, data templateData) error {
	entries, err := fs.ReadDir(srcFS, srcPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcEntryPath := path.Join(srcPath, entry.Name())
		destEntryPath := filepath.Join(destPath, entry.Name())

		if entry.IsDir() {
			if err := os.MkdirAll(destEntryPath, 0755); err != nil {
				return err
			}
			if err := copyTemplateDir(srcFS, srcEntryPath, destEntryPath, data); err != nil {
				return err
			}
		} else {
			if err := copyTemplateFile(srcFS, srcEntryPath, destEntryPath, data); err != nil {
				return err
			}
		}
	}

	return nil
}

func copyTemplateFile(srcFS fs.FS, srcPath, destPath string, data templateData) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	content, err := fs.ReadFile(srcFS, srcPath)
	if err != nil {
		return err
	}

	tmpl, err := template.New(srcPath).Parse(string(content))
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return err
	}

	return os.WriteFile(destPath, buf.Bytes(), 0644)
}

func createGoMod(projectName, targetDir string) error {
	moduleName := ValidateProjectNameForModule(projectName)
	modContent := fmt.Sprintf("module %s\n\ngo 1.23\n", moduleName)
	modPath := filepath.Join(targetDir, "go.mod")
	return os.WriteFile(modPath, []byte(modContent), 0644)
}

func runTemplGenerate(targetDir string) error {
	cmd := exec.Command("templ", "generate")
	cmd.Dir = targetDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("templ generate failed: %s", string(output))
	}
	return nil
}

func initGit(targetDir string) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = targetDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git init failed: %s", string(output))
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = targetDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed: %s", string(output))
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit from zt create")
	cmd.Dir = targetDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit failed: %s", string(output))
	}

	return nil
}

func cleanupOnError(targetDir string) {
	os.RemoveAll(targetDir)
}
