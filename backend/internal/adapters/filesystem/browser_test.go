package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

// findEntry returns the entry with the given name, or nil if absent.
func findEntry(entries []entryByName, name string) *entryByName {
	for i := range entries {
		if entries[i].name == name {
			return &entries[i]
		}
	}
	return nil
}

type entryByName struct {
	name      string
	isDir     bool
	isSymlink bool
	isBroken  bool
}

// setupTree builds a temp directory containing:
//   - realdir/        (a real directory)
//   - realfile.txt    (a real file)
//   - linkdir   -> realdir       (symlink to a directory)
//   - linkfile  -> realfile.txt  (symlink to a file)
//   - broken    -> nonexistent   (broken symlink)
//   - .hidden          (hidden file)
func setupTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	if err := os.Mkdir(filepath.Join(root, "realdir"), 0o755); err != nil {
		t.Fatalf("mkdir realdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "realfile.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatalf("write realfile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".hidden"), []byte("h"), 0o644); err != nil {
		t.Fatalf("write hidden: %v", err)
	}
	if err := os.Symlink(filepath.Join(root, "realdir"), filepath.Join(root, "linkdir")); err != nil {
		t.Fatalf("symlink linkdir: %v", err)
	}
	if err := os.Symlink(filepath.Join(root, "realfile.txt"), filepath.Join(root, "linkfile")); err != nil {
		t.Fatalf("symlink linkfile: %v", err)
	}
	if err := os.Symlink(filepath.Join(root, "does-not-exist"), filepath.Join(root, "broken")); err != nil {
		t.Fatalf("symlink broken: %v", err)
	}
	return root
}

func listForTest(t *testing.T, root string, showHidden bool) []entryByName {
	t.Helper()
	b := NewBrowser()
	got, err := b.ListDir(root, showHidden)
	if err != nil {
		t.Fatalf("ListDir: %v", err)
	}
	out := make([]entryByName, 0, len(got))
	for _, e := range got {
		out = append(out, entryByName{e.Name, e.IsDir, e.IsSymlink, e.IsBroken})
	}
	return out
}

func TestListDir_RealDirAndFile(t *testing.T) {
	entries := listForTest(t, setupTree(t), false)

	if d := findEntry(entries, "realdir"); d == nil {
		t.Fatal("realdir missing")
	} else if !d.isDir || d.isSymlink || d.isBroken {
		t.Errorf("realdir: want dir/non-symlink/non-broken, got %+v", *d)
	}

	if f := findEntry(entries, "realfile.txt"); f == nil {
		t.Fatal("realfile.txt missing")
	} else if f.isDir || f.isSymlink || f.isBroken {
		t.Errorf("realfile.txt: want file/non-symlink/non-broken, got %+v", *f)
	}
}

func TestListDir_SymlinkToDir_IsNavigableDir(t *testing.T) {
	entries := listForTest(t, setupTree(t), false)

	d := findEntry(entries, "linkdir")
	if d == nil {
		t.Fatal("linkdir missing")
	}
	if !d.isDir {
		t.Error("linkdir: expected IsDir=true (target is a directory)")
	}
	if !d.isSymlink {
		t.Error("linkdir: expected IsSymlink=true")
	}
	if d.isBroken {
		t.Error("linkdir: expected IsBroken=false")
	}
}

func TestListDir_SymlinkToFile_IsSymlinkNonDir(t *testing.T) {
	entries := listForTest(t, setupTree(t), false)

	f := findEntry(entries, "linkfile")
	if f == nil {
		t.Fatal("linkfile missing")
	}
	if f.isDir {
		t.Error("linkfile: expected IsDir=false (target is a file)")
	}
	if !f.isSymlink {
		t.Error("linkfile: expected IsSymlink=true")
	}
	if f.isBroken {
		t.Error("linkfile: expected IsBroken=false")
	}
}

func TestListDir_BrokenSymlink_FlaggedBroken(t *testing.T) {
	entries := listForTest(t, setupTree(t), false)

	b := findEntry(entries, "broken")
	if b == nil {
		t.Fatal("broken symlink missing from listing")
	}
	if !b.isSymlink {
		t.Error("broken: expected IsSymlink=true")
	}
	if !b.isBroken {
		t.Error("broken: expected IsBroken=true")
	}
	if b.isDir {
		t.Error("broken: expected IsDir=false")
	}
}

func TestListDir_HiddenFilteredByDefault(t *testing.T) {
	entries := listForTest(t, setupTree(t), false)
	if findEntry(entries, ".hidden") != nil {
		t.Error(".hidden should be filtered when showHidden=false")
	}
}

func TestListDir_HiddenShownWhenRequested(t *testing.T) {
	entries := listForTest(t, setupTree(t), true)
	if findEntry(entries, ".hidden") == nil {
		t.Error(".hidden should be present when showHidden=true")
	}
}

func TestListDir_DirsSortedBeforeFiles(t *testing.T) {
	entries := listForTest(t, setupTree(t), false)
	// All dir entries (incl. symlinked dirs) must precede non-dir entries.
	seenNonDir := false
	for _, e := range entries {
		if !e.isDir {
			seenNonDir = true
		} else if seenNonDir {
			t.Errorf("directory %q appeared after a non-directory entry", e.name)
		}
	}
}
