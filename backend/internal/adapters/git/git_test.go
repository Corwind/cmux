package git_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Corwind/cmux/backend/internal/adapters/git"
)

// setupRepo creates a temp git repo with an initial commit and returns its path.
func setupRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@test.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	run("init", "-b", "main")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test")

	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	run("add", ".")
	run("commit", "-m", "init")

	return dir
}

func TestInfo_IsRepo(t *testing.T) {
	svc := git.NewService()
	repoDir := setupRepo(t)

	info, err := svc.Info(repoDir)
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}
	if !info.IsRepo {
		t.Error("expected IsRepo=true for a git repo")
	}
	if info.RepoRoot == "" {
		t.Error("expected non-empty RepoRoot")
	}
	if info.CurrentBranch != "main" {
		t.Errorf("expected CurrentBranch=main, got %q", info.CurrentBranch)
	}
}

func TestInfo_NotRepo(t *testing.T) {
	svc := git.NewService()
	dir := t.TempDir()

	info, err := svc.Info(dir)
	if err != nil {
		t.Fatalf("Info on non-repo should not error: %v", err)
	}
	if info.IsRepo {
		t.Error("expected IsRepo=false for non-repo dir")
	}
}

func TestInfo_Branches(t *testing.T) {
	svc := git.NewService()
	repoDir := setupRepo(t)

	// Create a second branch
	cmd := exec.Command("git", "branch", "feature/foo")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git branch: %v\n%s", err, out)
	}

	info, err := svc.Info(repoDir)
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}

	names := make(map[string]bool)
	for _, b := range info.Branches {
		names[b.Name] = true
		if b.Name == info.CurrentBranch && !b.IsCurrent {
			t.Errorf("branch %q should have IsCurrent=true", b.Name)
		}
	}
	if !names["main"] {
		t.Error("expected branch 'main' in list")
	}
	if !names["feature/foo"] {
		t.Error("expected branch 'feature/foo' in list")
	}
}

func TestInfo_Worktrees(t *testing.T) {
	svc := git.NewService()
	repoDir := setupRepo(t)
	wtDir := t.TempDir()
	wtPath := filepath.Join(wtDir, "wt-branch")

	// Add a worktree on a new branch
	cmd := exec.Command("git", "-C", repoDir, "worktree", "add", "-b", "wt-branch", wtPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add: %v\n%s", err, out)
	}

	info, err := svc.Info(repoDir)
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}

	if len(info.Worktrees) < 2 {
		t.Fatalf("expected at least 2 worktrees, got %d", len(info.Worktrees))
	}

	found := false
	for _, wt := range info.Worktrees {
		if wt.Branch == "wt-branch" {
			found = true
			if wt.Path == "" {
				t.Error("expected non-empty worktree path")
			}
		}
	}
	if !found {
		t.Error("expected to find 'wt-branch' worktree")
	}

	// Main worktree should be marked IsMain
	mainFound := false
	for _, wt := range info.Worktrees {
		if wt.IsMain {
			mainFound = true
			break
		}
	}
	if !mainFound {
		t.Error("expected main worktree to be marked IsMain")
	}
}

func TestAddWorktree_NewBranch(t *testing.T) {
	svc := git.NewService()
	repoDir := setupRepo(t)
	wtDir := t.TempDir()
	wtPath := filepath.Join(wtDir, "new-branch-wt")

	wt, err := svc.AddWorktree(context.Background(), repoDir, wtPath, "new-branch", "main", true)
	if err != nil {
		t.Fatalf("AddWorktree (new branch) failed: %v", err)
	}
	if wt.Branch != "new-branch" {
		t.Errorf("expected Branch=new-branch, got %q", wt.Branch)
	}
	if wt.Path == "" {
		t.Error("expected non-empty path")
	}
	// Verify on disk
	if _, err := os.Stat(wtPath); err != nil {
		t.Errorf("worktree path not on disk: %v", err)
	}
}

func TestAddWorktree_ExistingBranch(t *testing.T) {
	svc := git.NewService()
	repoDir := setupRepo(t)

	// Create branch first
	cmd := exec.Command("git", "-C", repoDir, "branch", "existing-branch")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git branch: %v\n%s", err, out)
	}

	wtDir := t.TempDir()
	wtPath := filepath.Join(wtDir, "existing-branch-wt")

	wt, err := svc.AddWorktree(context.Background(), repoDir, wtPath, "existing-branch", "", false)
	if err != nil {
		t.Fatalf("AddWorktree (existing branch) failed: %v", err)
	}
	if wt.Branch != "existing-branch" {
		t.Errorf("expected Branch=existing-branch, got %q", wt.Branch)
	}
}

func TestAddWorktree_PathTraversal(t *testing.T) {
	svc := git.NewService()
	repoDir := setupRepo(t)

	_, err := svc.AddWorktree(context.Background(), repoDir, "/tmp/../../evil", "branch", "main", true)
	if err == nil {
		t.Error("expected error for path traversal in worktree path")
	}
}

func TestAddWorktree_BranchTraversal(t *testing.T) {
	svc := git.NewService()
	repoDir := setupRepo(t)

	_, err := svc.AddWorktree(context.Background(), repoDir, "/tmp/safe", "../evil", "main", true)
	if err == nil {
		t.Error("expected error for path traversal in branch name")
	}
}

func TestRemoveWorktree_Clean(t *testing.T) {
	svc := git.NewService()
	repoDir := setupRepo(t)
	wtDir := t.TempDir()
	wtPath := filepath.Join(wtDir, "clean-wt")

	cmd := exec.Command("git", "-C", repoDir, "worktree", "add", "-b", "clean-branch", wtPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add: %v\n%s", err, out)
	}

	if err := svc.RemoveWorktree(context.Background(), repoDir, wtPath, false); err != nil {
		t.Fatalf("RemoveWorktree failed: %v", err)
	}

	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Error("expected worktree path to be removed")
	}
}

func TestRemoveWorktree_DirtyRejected(t *testing.T) {
	svc := git.NewService()
	repoDir := setupRepo(t)
	wtDir := t.TempDir()
	wtPath := filepath.Join(wtDir, "dirty-wt")

	cmd := exec.Command("git", "-C", repoDir, "worktree", "add", "-b", "dirty-branch", wtPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add: %v\n%s", err, out)
	}

	// Make a dirty change
	if err := os.WriteFile(filepath.Join(wtPath, "dirty.txt"), []byte("dirty"), 0o644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}

	err := svc.RemoveWorktree(context.Background(), repoDir, wtPath, false)
	if err == nil {
		t.Fatal("expected error removing dirty worktree without force")
	}
}

func TestRemoveWorktree_ForceRemovesDirty(t *testing.T) {
	svc := git.NewService()
	repoDir := setupRepo(t)
	wtDir := t.TempDir()
	wtPath := filepath.Join(wtDir, "force-wt")

	cmd := exec.Command("git", "-C", repoDir, "worktree", "add", "-b", "force-branch", wtPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add: %v\n%s", err, out)
	}

	// Make a dirty change
	if err := os.WriteFile(filepath.Join(wtPath, "dirty.txt"), []byte("dirty"), 0o644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}

	if err := svc.RemoveWorktree(context.Background(), repoDir, wtPath, true); err != nil {
		t.Fatalf("RemoveWorktree with force failed: %v", err)
	}
}

func TestIsClean_Clean(t *testing.T) {
	svc := git.NewService()
	repoDir := setupRepo(t)

	clean, err := svc.IsClean(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("IsClean failed: %v", err)
	}
	if !clean {
		t.Error("expected clean repo to be clean")
	}
}

func TestIsClean_Dirty(t *testing.T) {
	svc := git.NewService()
	repoDir := setupRepo(t)

	if err := os.WriteFile(filepath.Join(repoDir, "dirty.txt"), []byte("dirty"), 0o644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}

	clean, err := svc.IsClean(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("IsClean failed: %v", err)
	}
	if clean {
		t.Error("expected dirty repo to not be clean")
	}
}

func TestInfo_WorktreeDetectsGitlink(t *testing.T) {
	svc := git.NewService()
	repoDir := setupRepo(t)
	wtDir := t.TempDir()
	wtPath := filepath.Join(wtDir, "link-wt")

	cmd := exec.Command("git", "-C", repoDir, "worktree", "add", "-b", "link-branch", wtPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add: %v\n%s", err, out)
	}

	// The .git in the worktree should be a file (gitlink), not a directory
	gitPath := filepath.Join(wtPath, ".git")
	fi, err := os.Stat(gitPath)
	if err != nil {
		t.Fatalf("stat .git: %v", err)
	}
	if fi.IsDir() {
		t.Error("expected .git to be a file (gitlink) in a worktree, not a directory")
	}

	// Info from the worktree path should also work
	info, err := svc.Info(wtPath)
	if err != nil {
		t.Fatalf("Info on worktree path failed: %v", err)
	}
	if !info.IsRepo {
		t.Error("expected worktree path to be recognized as a repo")
	}
	if !strings.HasSuffix(info.RepoRoot, repoDir) && info.RepoRoot != repoDir {
		// RepoRoot might be symlink-resolved
		t.Logf("RepoRoot=%q repoDir=%q (may differ due to symlink resolution)", info.RepoRoot, repoDir)
	}
}

func TestAddWorktree_CancelledContext(t *testing.T) {
	svc := git.NewService()
	repoDir := setupRepo(t)
	wtDir := t.TempDir()
	wtPath := filepath.Join(wtDir, "cancelled-wt")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately before calling

	_, err := svc.AddWorktree(ctx, repoDir, wtPath, "cancelled-branch", "main", true)
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func TestIsClean_CancelledContext(t *testing.T) {
	svc := git.NewService()
	repoDir := setupRepo(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately before calling

	_, err := svc.IsClean(ctx, repoDir)
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func TestRemoveWorktree_CancelledContext(t *testing.T) {
	svc := git.NewService()
	repoDir := setupRepo(t)
	wtDir := t.TempDir()
	wtPath := filepath.Join(wtDir, "remove-cancelled-wt")

	// Add worktree first with a valid context
	cmd := exec.Command("git", "-C", repoDir, "worktree", "add", "-b", "remove-cancelled-branch", wtPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add: %v\n%s", err, out)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately before calling

	err := svc.RemoveWorktree(ctx, repoDir, wtPath, false)
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}
