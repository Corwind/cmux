package config

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Corwind/cmux/backend/internal/domain"
)

// ResolveShellEnv builds the environment variable slice for spawned processes
// based on the provided config. It captures the shell's login environment,
// overlays explicit env vars from config, and ensures essential vars are set.
func ResolveShellEnv(cfg domain.Config) []string {
	// If nothing is configured, pass through the current environment
	if cfg.Shell.Path == "" && len(cfg.Shell.InitFiles) == 0 && len(cfg.Env) == 0 {
		return os.Environ()
	}

	shell := cfg.Shell.Path
	if shell == "" {
		shell = os.Getenv("SHELL")
	}
	if shell == "" {
		shell = "/bin/zsh"
	}

	var base []string
	var err error

	if len(cfg.Shell.InitFiles) > 0 {
		base, err = captureWithInitFiles(shell, cfg.Shell.InitFiles)
	} else {
		base, err = captureLoginEnv(shell)
	}

	if err != nil {
		log.Printf("warning: could not capture shell env: %v (falling back to process env)", err)
		base = os.Environ()
	}

	// Overlay explicit env vars from config
	if len(cfg.Env) > 0 {
		overlay := mapToSlice(cfg.Env)
		base = MergeEnv(base, overlay)
	}

	// Ensure essential vars are present
	base = MergeEnv(base, []string{
		"TERM=xterm-256color",
		"LANG=en_US.UTF-8",
	})

	return base
}

// MergeEnv merges overlay env vars onto a base, with overlay taking precedence.
// Both base and overlay are slices of KEY=VALUE strings.
func MergeEnv(base, overlay []string) []string {
	index := make(map[string]int, len(base))
	result := make([]string, len(base))
	copy(result, base)

	for i, entry := range result {
		if key, _, ok := strings.Cut(entry, "="); ok {
			index[key] = i
		}
	}

	for _, entry := range overlay {
		key, _, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if idx, exists := index[key]; exists {
			result[idx] = entry
		} else {
			index[key] = len(result)
			result = append(result, entry)
		}
	}

	return result
}

func captureLoginEnv(shell string) ([]string, error) {
	cmd := exec.Command(shell, "-li", "-c", "env")
	cmd.Stdin = nil

	out, err := runWithTimeout(cmd, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("capture login env from %s: %w", shell, err)
	}

	return parseEnvOutput(string(out)), nil
}

func captureWithInitFiles(shell string, initFiles []string) ([]string, error) {
	var parts []string
	for _, f := range initFiles {
		parts = append(parts, "source "+shellQuote(f))
	}
	parts = append(parts, "env")
	script := strings.Join(parts, " && ")

	cmd := exec.Command(shell, "-c", script)
	cmd.Stdin = nil

	out, err := runWithTimeout(cmd, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("capture env with init files: %w", err)
	}

	return parseEnvOutput(string(out)), nil
}

func parseEnvOutput(output string) []string {
	var env []string
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if _, _, ok := strings.Cut(line, "="); ok {
			env = append(env, line)
		}
	}
	return env
}

// shellQuote wraps a string in single quotes, escaping any embedded single quotes.
// This prevents shell metacharacter injection when building shell commands.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

func mapToSlice(m map[string]string) []string {
	s := make([]string, 0, len(m))
	for k, v := range m {
		s = append(s, k+"="+v)
	}
	return s
}

func runWithTimeout(cmd *exec.Cmd, timeout time.Duration) ([]byte, error) {
	type result struct {
		out []byte
		err error
	}
	ch := make(chan result, 1)
	go func() {
		out, err := cmd.Output()
		ch <- result{out, err}
	}()

	select {
	case r := <-ch:
		return r.out, r.err
	case <-time.After(timeout):
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return nil, fmt.Errorf("command timed out after %s", timeout)
	}
}
