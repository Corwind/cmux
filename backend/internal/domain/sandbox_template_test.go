package domain

import (
	"strings"
	"testing"
)

func TestRulesToSBPL_SingleRule(t *testing.T) {
	rules := []SandboxRule{
		{Path: "/tmp", Read: true, Write: true, Metadata: false},
	}
	sbpl := RulesToSBPL(rules)

	if !strings.Contains(sbpl, `(allow file-read* (subpath "/tmp"))`) {
		t.Error("missing file-read* rule")
	}
	if !strings.Contains(sbpl, `(allow file-write* (subpath "/tmp"))`) {
		t.Error("missing file-write* rule")
	}
	if strings.Contains(sbpl, "file-read-metadata") {
		t.Error("should not contain metadata rule when Metadata is false")
	}
}

func TestRulesToSBPL_MultipleRules(t *testing.T) {
	rules := []SandboxRule{
		{Path: "/tmp", Read: true, Write: true},
		{Path: "/opt/homebrew", Read: true, Write: false},
	}
	sbpl := RulesToSBPL(rules)

	if !strings.Contains(sbpl, `(allow file-read* (subpath "/opt/homebrew"))`) {
		t.Error("missing homebrew read rule")
	}
	if strings.Contains(sbpl, `(allow file-write* (subpath "/opt/homebrew"))`) {
		t.Error("should not contain homebrew write rule")
	}
}

func TestRulesToSBPL_MetadataOnly(t *testing.T) {
	rules := []SandboxRule{
		{Path: "/var/log", Metadata: true},
	}
	sbpl := RulesToSBPL(rules)

	if !strings.Contains(sbpl, `(allow file-read-metadata (subpath "/var/log"))`) {
		t.Error("missing metadata rule")
	}
	if strings.Contains(sbpl, "file-read*") {
		t.Error("should not contain file-read* rule")
	}
}

func TestRulesToSBPL_Empty(t *testing.T) {
	sbpl := RulesToSBPL(nil)
	if sbpl != "" {
		t.Errorf("expected empty string, got %q", sbpl)
	}
}

func TestSBPLToRules_Basic(t *testing.T) {
	content := `(allow file-read* (subpath "/tmp"))
(allow file-write* (subpath "/tmp"))`

	rules := SBPLToRules(content)
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Path != "/tmp" {
		t.Errorf("expected path /tmp, got %q", rules[0].Path)
	}
	if !rules[0].Read {
		t.Error("expected Read=true")
	}
	if !rules[0].Write {
		t.Error("expected Write=true")
	}
	if rules[0].Metadata {
		t.Error("expected Metadata=false")
	}
}

func TestSBPLToRules_MultiplePaths(t *testing.T) {
	content := `(allow file-read* (subpath "/tmp"))
(allow file-write* (subpath "/tmp"))
(allow file-read* (subpath "/opt/homebrew"))
(allow file-read-metadata (subpath "/var/log"))`

	rules := SBPLToRules(content)
	if len(rules) != 3 {
		t.Fatalf("expected 3 rules, got %d", len(rules))
	}

	// Check order is preserved
	if rules[0].Path != "/tmp" {
		t.Errorf("expected first path /tmp, got %q", rules[0].Path)
	}
	if rules[1].Path != "/opt/homebrew" {
		t.Errorf("expected second path /opt/homebrew, got %q", rules[1].Path)
	}
	if rules[2].Path != "/var/log" {
		t.Errorf("expected third path /var/log, got %q", rules[2].Path)
	}

	if !rules[1].Read || rules[1].Write {
		t.Error("homebrew should be read-only")
	}
	if !rules[2].Metadata || rules[2].Read || rules[2].Write {
		t.Error("/var/log should be metadata-only")
	}
}

func TestSBPLToRules_IgnoresComments(t *testing.T) {
	content := `;; some comment
(allow file-read* (subpath "/tmp"))
;; another comment`

	rules := SBPLToRules(content)
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
}

func TestSBPLToRules_IgnoresNonFileRules(t *testing.T) {
	content := `(allow process-exec*)
(allow file-read* (subpath "/tmp"))
(allow network-outbound)`

	rules := SBPLToRules(content)
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule (only file rules), got %d", len(rules))
	}
}

func TestSBPLToRules_Empty(t *testing.T) {
	rules := SBPLToRules("")
	if len(rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(rules))
	}
}

func TestRoundTrip(t *testing.T) {
	original := []SandboxRule{
		{Path: "/tmp", Read: true, Write: true},
		{Path: "/opt/homebrew", Read: true},
		{Path: "/var/log", Metadata: true},
	}

	sbpl := RulesToSBPL(original)
	parsed := SBPLToRules(sbpl)

	if len(parsed) != len(original) {
		t.Fatalf("roundtrip: expected %d rules, got %d", len(original), len(parsed))
	}

	for i, o := range original {
		p := parsed[i]
		if o.Path != p.Path || o.Read != p.Read || o.Write != p.Write || o.Metadata != p.Metadata {
			t.Errorf("roundtrip mismatch at index %d: original=%+v parsed=%+v", i, o, p)
		}
	}
}

func TestSBPLToRules_WithParamSyntax(t *testing.T) {
	// Lines with (param ...) syntax should be ignored (not subpath "literal")
	content := `(allow file-write* (subpath (param "WORKING_DIR")))
(allow file-read* (subpath "/tmp"))`

	rules := SBPLToRules(content)
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule (param syntax ignored), got %d", len(rules))
	}
	if rules[0].Path != "/tmp" {
		t.Errorf("expected path /tmp, got %q", rules[0].Path)
	}
}
