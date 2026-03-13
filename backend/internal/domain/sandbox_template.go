package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type SandboxTemplate struct {
	ID        string
	Name      string
	Content   string // Raw SBPL fragment (allow rules only)
	IsDefault bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewSandboxTemplate(name, content string) (SandboxTemplate, error) {
	if name == "" {
		return SandboxTemplate{}, fmt.Errorf("template name cannot be empty")
	}
	if content == "" {
		return SandboxTemplate{}, fmt.Errorf("template content cannot be empty")
	}

	if err := ValidateTemplateContent(content); err != nil {
		return SandboxTemplate{}, err
	}

	now := time.Now()
	return SandboxTemplate{
		ID:        uuid.New().String(),
		Name:      name,
		Content:   content,
		IsDefault: false,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// ValidateTemplateContent checks that SBPL content doesn't contain
// forbidden directives (version, deny).
func ValidateTemplateContent(content string) error {
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
