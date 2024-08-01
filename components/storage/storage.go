package storage

type Storage interface {
	CreateDir(name string) error
	ListDir(name string) ([]*FileInfo, error)

	CreateFile(name string, fileData []byte, options ...Option) error
	GetFileInfo(name string) (*FileInfo, error)
	GetFileData(name string) ([]byte, error)

	GetFullName(name string) (string, error)

	DirExists(name string) (bool, error)
	FileExists(name string) (bool, error)
	Remove(name string, options ...Option) error
}

type Options struct {
	ForceReplace  bool
	AutoCreateDir bool
	RemoveAll     bool
}
type Option func(options *Options) *Options

func WithForceReplace(options *Options) *Options {
	options.ForceReplace = true
	return options
}

func WithAutoCreateDir(options *Options) *Options {
	options.AutoCreateDir = true
	return options
}

func WithRemoveAll(options *Options) *Options {
	options.RemoveAll = true
	return options
}

type FileInfo struct {
	IsDir      bool
	Path       string
	FileSize   int64
	UpdateTime int64
}
