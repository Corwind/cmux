package config

import (
	"os"
	"testing"

	"github.com/Corwind/cmux/backend/internal/domain"
)

func TestMergeEnvOverwritesExisting(t *testing.T) {
	base := []string{"FOO=old", "BAR=keep"}
	overlay := []string{"FOO=new"}

	result := MergeEnv(base, overlay)

	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d: %v", len(result), result)
	}
	if result[0] != "FOO=new" {
		t.Errorf("expected FOO=new, got %q", result[0])
	}
	if result[1] != "BAR=keep" {
		t.Errorf("expected BAR=keep, got %q", result[1])
	}
}

func TestMergeEnvAppendsNew(t *testing.T) {
	base := []string{"FOO=bar"}
	overlay := []string{"NEW=val"}

	result := MergeEnv(base, overlay)

	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d: %v", len(result), result)
	}
	if result[0] != "FOO=bar" {
		t.Errorf("expected FOO=bar, got %q", result[0])
	}
	if result[1] != "NEW=val" {
		t.Errorf("expected NEW=val, got %q", result[1])
	}
}

func TestMergeEnvEmptyBase(t *testing.T) {
	result := MergeEnv(nil, []string{"FOO=bar"})

	if len(result) != 1 || result[0] != "FOO=bar" {
		t.Errorf("expected [FOO=bar], got %v", result)
	}
}

func TestMergeEnvEmptyOverlay(t *testing.T) {
	base := []string{"FOO=bar"}
	result := MergeEnv(base, nil)

	if len(result) != 1 || result[0] != "FOO=bar" {
		t.Errorf("expected [FOO=bar], got %v", result)
	}
}

func TestMergeEnvBothEmpty(t *testing.T) {
	result := MergeEnv(nil, nil)

	if len(result) != 0 {
		t.Errorf("expected empty result, got %v", result)
	}
}

func TestMergeEnvSkipsInvalidOverlay(t *testing.T) {
	base := []string{"FOO=bar"}
	overlay := []string{"NOEQUALSSIGN"}

	result := MergeEnv(base, overlay)

	if len(result) != 1 || result[0] != "FOO=bar" {
		t.Errorf("expected [FOO=bar], got %v", result)
	}
}

func TestParseEnvOutput(t *testing.T) {
	input := "HOME=/Users/test\nPATH=/usr/bin:/bin\nSHELL=/bin/zsh\nnotanenvar\n"

	result := parseEnvOutput(input)

	if len(result) != 3 {
		t.Fatalf("expected 3 entries, got %d: %v", len(result), result)
	}
	if result[0] != "HOME=/Users/test" {
		t.Errorf("expected HOME=/Users/test, got %q", result[0])
	}
	if result[1] != "PATH=/usr/bin:/bin" {
		t.Errorf("expected PATH=/usr/bin:/bin, got %q", result[1])
	}
}

func TestParseEnvOutputEmpty(t *testing.T) {
	result := parseEnvOutput("")
	if len(result) != 0 {
		t.Errorf("expected empty result, got %v", result)
	}
}

func TestResolveShellEnvEmptyConfig(t *testing.T) {
	cfg := domain.Config{}

	result := ResolveShellEnv(cfg)

	// Should return os.Environ() when nothing is configured
	osEnv := os.Environ()
	if len(result) != len(osEnv) {
		t.Errorf("expected %d env vars (os.Environ()), got %d", len(osEnv), len(result))
	}
}

func TestResolveShellEnvWithExplicitEnvMap(t *testing.T) {
	cfg := domain.Config{
		Env: map[string]string{
			"CUSTOM_VAR": "custom_value",
		},
	}

	result := ResolveShellEnv(cfg)

	found := false
	for _, entry := range result {
		if entry == "CUSTOM_VAR=custom_value" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected CUSTOM_VAR=custom_value in resolved env")
	}

	// TERM and LANG should always be present
	foundTerm := false
	foundLang := false
	for _, entry := range result {
		if entry == "TERM=xterm-256color" {
			foundTerm = true
		}
		if entry == "LANG=en_US.UTF-8" {
			foundLang = true
		}
	}
	if !foundTerm {
		t.Error("expected TERM=xterm-256color in resolved env")
	}
	if !foundLang {
		t.Error("expected LANG=en_US.UTF-8 in resolved env")
	}
}

func TestShellQuote(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "'simple'"},
		{"with space", "'with space'"},
		{"it's", "'it'\"'\"'s'"},
		{"", "''"},
		{"/path/to/file", "'/path/to/file'"},
		{"$(dangerous)", "'$(dangerous)'"},
		{"; rm -rf /", "'; rm -rf /'"},
	}

	for _, tc := range tests {
		got := shellQuote(tc.input)
		if got != tc.want {
			t.Errorf("shellQuote(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestMapToSlice(t *testing.T) {
	m := map[string]string{"A": "1", "B": "2"}
	result := mapToSlice(m)

	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}

	found := map[string]bool{}
	for _, entry := range result {
		found[entry] = true
	}
	if !found["A=1"] || !found["B=2"] {
		t.Errorf("expected A=1 and B=2, got %v", result)
	}
}
