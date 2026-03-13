package sandbox

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Template represents a loaded SBPL template fragment.
type Template struct {
	Name    string
	Content string
}

// LoadTemplate loads a .sbpl file from the template directory and validates it.
func (pb *ProfileBuilder) LoadTemplate(name string) (Template, error) {
	path := filepath.Join(pb.templateDir, name+".sbpl")

	data, err := os.ReadFile(path)
	if err != nil {
		return Template{}, fmt.Errorf("load template %q: %w", name, err)
	}

	content := string(data)

	if err := validateTemplate(content); err != nil {
		return Template{}, fmt.Errorf("validate template %q: %w", name, err)
	}

	return Template{
		Name:    name,
		Content: strings.TrimSpace(content),
	}, nil
}

// ListTemplates returns the names of all available .sbpl templates.
func (pb *ProfileBuilder) ListTemplates() ([]string, error) {
	entries, err := os.ReadDir(pb.templateDir)
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if filepath.Ext(e.Name()) == ".sbpl" {
			names = append(names, strings.TrimSuffix(e.Name(), ".sbpl"))
		}
	}

	return names, nil
}

func validateTemplate(content string) error {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "(version ") {
			return fmt.Errorf("template must not contain (version) directive")
		}
		if strings.HasPrefix(trimmed, "(deny ") {
			return fmt.Errorf("template must not contain (deny) directive")
		}
	}
	return nil
}
