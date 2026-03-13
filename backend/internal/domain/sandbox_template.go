package domain

import (
	"fmt"
	"regexp"
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

// SandboxRule represents a structured file access rule for a path.
type SandboxRule struct {
	Path     string `json:"path"`
	Read     bool   `json:"read"`
	Write    bool   `json:"write"`
	Metadata bool   `json:"metadata"`
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

// RulesToSBPL converts structured rules to SBPL content lines.
func RulesToSBPL(rules []SandboxRule) string {
	var lines []string
	for _, r := range rules {
		if r.Read {
			lines = append(lines, fmt.Sprintf(`(allow file-read* (subpath "%s"))`, r.Path))
		}
		if r.Write {
			lines = append(lines, fmt.Sprintf(`(allow file-write* (subpath "%s"))`, r.Path))
		}
		if r.Metadata {
			lines = append(lines, fmt.Sprintf(`(allow file-read-metadata (subpath "%s"))`, r.Path))
		}
	}
	return strings.Join(lines, "\n")
}

var sbplRulePattern = regexp.MustCompile(`^\(allow (file-read\*|file-write\*|file-read-metadata) \(subpath "([^"]+)"\)\)$`)

// SBPLToRules parses SBPL content back to structured rules, grouping by path.
func SBPLToRules(content string) []SandboxRule {
	ruleMap := make(map[string]*SandboxRule)
	var order []string

	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		matches := sbplRulePattern.FindStringSubmatch(trimmed)
		if matches == nil {
			continue
		}

		action := matches[1]
		path := matches[2]

		rule, exists := ruleMap[path]
		if !exists {
			rule = &SandboxRule{Path: path}
			ruleMap[path] = rule
			order = append(order, path)
		}

		switch action {
		case "file-read*":
			rule.Read = true
		case "file-write*":
			rule.Write = true
		case "file-read-metadata":
			rule.Metadata = true
		}
	}

	rules := make([]SandboxRule, 0, len(order))
	for _, path := range order {
		rules = append(rules, *ruleMap[path])
	}
	return rules
}
