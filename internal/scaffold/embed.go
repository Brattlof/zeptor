package scaffold

import (
	"embed"
	"io/fs"
)

//go:embed templates/minimal templates/basic templates/api
//go:embed templates/minimal/.gitignore templates/basic/.gitignore templates/api/.gitignore
var templatesFS embed.FS

func GetTemplates(templateName string) (fs.FS, error) {
	return fs.Sub(templatesFS, "templates/"+templateName)
}

func TemplateExists(name string) bool {
	entries, err := templatesFS.ReadDir("templates")
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.Name() == name {
			return true
		}
	}
	return false
}

func AvailableTemplates() []string {
	entries, err := templatesFS.ReadDir("templates")
	if err != nil {
		return nil
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	return names
}
