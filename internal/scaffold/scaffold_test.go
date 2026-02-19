package scaffold

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateProjectName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple", "myapp", false},
		{"valid with dash", "my-app", false},
		{"valid with underscore", "my_app", false},
		{"valid alphanumeric", "app123", false},
		{"empty", "", true},
		{"too long", "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz", true},
		{"starts with dash", "-app", true},
		{"starts with underscore", "_app", true},
		{"uppercase", "MyApp", true},
		{"with space", "my app", true},
		{"with special char", "my@app", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProjectName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProjectName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateProjectNameForModule(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"my-app", "myapp"},
		{"my-project-name", "myprojectname"},
		{"simple", "simple"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ValidateProjectNameForModule(tt.input)
			if got != tt.expect {
				t.Errorf("ValidateProjectNameForModule(%q) = %q, want %q", tt.input, got, tt.expect)
			}
		})
	}
}

func TestTemplateExists(t *testing.T) {
	tests := []struct {
		name     string
		template string
		want     bool
	}{
		{"minimal exists", "minimal", true},
		{"basic exists", "basic", true},
		{"api exists", "api", true},
		{"nonexistent", "nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TemplateExists(tt.template)
			if got != tt.want {
				t.Errorf("TemplateExists(%q) = %v, want %v", tt.template, got, tt.want)
			}
		})
	}
}

func TestAvailableTemplates(t *testing.T) {
	templates := AvailableTemplates()
	if len(templates) < 3 {
		t.Errorf("AvailableTemplates() returned %d templates, want at least 3", len(templates))
	}

	found := make(map[string]bool)
	for _, tmpl := range templates {
		found[tmpl] = true
	}

	for _, expected := range []string{"minimal", "basic", "api"} {
		if !found[expected] {
			t.Errorf("AvailableTemplates() missing template %q", expected)
		}
	}
}

func TestCreateMinimalProject(t *testing.T) {
	tmpDir := t.TempDir()
	projectName := "test-project"
	targetDir := filepath.Join(tmpDir, projectName)

	opts := Options{
		ProjectName: projectName,
		Template:    "minimal",
		Port:        3000,
		SkipGit:     true,
		SkipTempl:   true,
		OutputDir:   targetDir,
	}

	if err := Create(opts); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	expectedFiles := []string{
		"app/page.templ",
		"zeptor.config.yaml",
		".gitignore",
		"README.md",
		"go.mod",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(targetDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s not created", file)
		}
	}

	content, err := os.ReadFile(filepath.Join(targetDir, "go.mod"))
	if err != nil {
		t.Fatalf("failed to read go.mod: %v", err)
	}
	if string(content) != "module testproject\n\ngo 1.23\n" {
		t.Errorf("go.mod content = %q, want module testproject", string(content))
	}
}

func TestCreateBasicProject(t *testing.T) {
	tmpDir := t.TempDir()
	projectName := "basic-app"
	targetDir := filepath.Join(tmpDir, projectName)

	opts := Options{
		ProjectName: projectName,
		Template:    "basic",
		Port:        8080,
		SkipGit:     true,
		SkipTempl:   true,
		OutputDir:   targetDir,
	}

	if err := Create(opts); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	expectedFiles := []string{
		"app/page.templ",
		"app/about/page.templ",
		"app/slug_/page.templ",
		"app/api/hello/route.go",
		"zeptor.config.yaml",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(targetDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s not created", file)
		}
	}

	content, err := os.ReadFile(filepath.Join(targetDir, "zeptor.config.yaml"))
	if err != nil {
		t.Fatalf("failed to read zeptor.config.yaml: %v", err)
	}
	configStr := string(content)
	if configStr == "" {
		t.Error("zeptor.config.yaml is empty")
	}
}

func TestCreateAPIProject(t *testing.T) {
	tmpDir := t.TempDir()
	projectName := "api-server"
	targetDir := filepath.Join(tmpDir, projectName)

	opts := Options{
		ProjectName: projectName,
		Template:    "api",
		Port:        3000,
		SkipGit:     true,
		SkipTempl:   true,
		OutputDir:   targetDir,
	}

	if err := Create(opts); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	expectedFiles := []string{
		"app/api/hello/route.go",
		"app/api/users/route.go",
		"zeptor.config.yaml",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(targetDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s not created", file)
		}
	}
}

func TestCreateInvalidTemplateName(t *testing.T) {
	opts := Options{
		ProjectName: "test-project",
		Template:    "invalid",
		Port:        3000,
		SkipGit:     true,
		SkipTempl:   true,
		OutputDir:   t.TempDir() + "/test",
	}

	err := Create(opts)
	if err == nil {
		t.Error("Create() expected error for invalid template, got nil")
	}
}

func TestCreateInvalidProjectName(t *testing.T) {
	opts := Options{
		ProjectName: "",
		Template:    "minimal",
		Port:        3000,
		SkipGit:     true,
		SkipTempl:   true,
		OutputDir:   t.TempDir() + "/test",
	}

	err := Create(opts)
	if err == nil {
		t.Error("Create() expected error for empty project name, got nil")
	}
}

func TestCreateDirectoryExists(t *testing.T) {
	tmpDir := t.TempDir()
	projectName := "existing-dir"
	targetDir := filepath.Join(tmpDir, projectName)

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}

	opts := Options{
		ProjectName: projectName,
		Template:    "minimal",
		Port:        3000,
		SkipGit:     true,
		SkipTempl:   true,
		OutputDir:   targetDir,
	}

	err := Create(opts)
	if err == nil {
		t.Error("Create() expected error when directory exists, got nil")
	}
}

func TestTemplateSubstitution(t *testing.T) {
	tmpDir := t.TempDir()
	projectName := "sub-test"
	targetDir := filepath.Join(tmpDir, projectName)

	opts := Options{
		ProjectName: projectName,
		Template:    "minimal",
		Port:        4000,
		SkipGit:     true,
		SkipTempl:   true,
		OutputDir:   targetDir,
	}

	if err := Create(opts); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	configContent, err := os.ReadFile(filepath.Join(targetDir, "zeptor.config.yaml"))
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	configStr := string(configContent)
	if configStr == "" {
		t.Error("config is empty")
	}
}
