package ports

type DirEntry struct {
	Name string
	// IsDir reflects the resolved target: true for a real directory or a
	// symlink whose target is a directory.
	IsDir bool
	// IsSymlink is true when the entry itself is a symbolic link.
	IsSymlink bool
	// IsBroken is true for a symlink whose target cannot be resolved.
	IsBroken bool
}

type FileBrowser interface {
	ListDir(path string, showHidden bool) ([]DirEntry, error)
	HomeDir() (string, error)
}
