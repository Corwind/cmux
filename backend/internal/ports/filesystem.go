package ports

type DirEntry struct {
	Name  string
	IsDir bool
}

type FileBrowser interface {
	ListDir(path string) ([]DirEntry, error)
	HomeDir() (string, error)
}
