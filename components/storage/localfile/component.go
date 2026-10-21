package localfile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/puper/leo/components/storage"
)

func New(config *Config) (*Component, error) {
	config.RootDir = filepath.Clean(config.RootDir)
	var err error
	config.RootDir, err = filepath.Abs(config.RootDir)
	if err != nil {
		return nil, err
	}
	return &Component{
		config: config,
	}, nil
}

var _ storage.Storage = (*Component)(nil)

// LocalStorage 结构体实现了 Storage 接口
type Component struct {
	config *Config
}

// 实现 Storage 接口的方法

// openRoot 在实际文件系统操作中限制路径，避免符号链接绕过词法校验。
func (me *Component) openRoot(name string, create bool) (*os.Root, string, error) {
	fullName, err := me.GetFullName(name)
	if err != nil {
		return nil, "", err
	}
	rel, err := filepath.Rel(me.config.RootDir, fullName)
	if err != nil {
		return nil, "", err
	}
	if create {
		if err := os.MkdirAll(me.config.RootDir, 0755); err != nil {
			return nil, "", err
		}
	}
	root, err := os.OpenRoot(me.config.RootDir)
	return root, rel, err
}

// CreateDir 创建目录。
func (me *Component) CreateDir(name string) error {
	root, rel, err := me.openRoot(name, true)
	if err != nil {
		return err
	}
	defer root.Close()
	return root.MkdirAll(rel, 0755)
}

func (me *Component) Remove(name string, options ...storage.Option) error {
	opts := &storage.Options{}
	for _, opt := range options {
		opt(opts)
	}
	root, rel, err := me.openRoot(name, false)
	if err != nil {
		if opts.RemoveAll && os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer root.Close()
	if rel == "." {
		// 配置的根路径本身是可信输入；先释放句柄，保留原有删除根目录的行为。
		if err := root.Close(); err != nil {
			return err
		}
		if opts.RemoveAll {
			return os.RemoveAll(me.config.RootDir)
		}
		return os.Remove(me.config.RootDir)
	}
	if opts.RemoveAll {
		return root.RemoveAll(rel)
	}
	return root.Remove(rel)
}

// ListDir 列出目录中的文件和子目录。
func (me *Component) ListDir(name string) ([]*storage.FileInfo, error) {
	root, rel, err := me.openRoot(name, false)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	files, err := fs.ReadDir(root.FS(), filepath.ToSlash(rel))
	if err != nil {
		return nil, err
	}
	var infos []*storage.FileInfo
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			return nil, err
		}
		infos = append(infos, &storage.FileInfo{
			IsDir:    info.IsDir(),
			Path:     filepath.Join(me.config.RootDir, rel, file.Name()),
			FileSize: info.Size(), UpdateTime: info.ModTime().Unix(),
		})
	}
	return infos, nil
}

func (me *Component) stat(name string) (os.FileInfo, error) {
	root, rel, err := me.openRoot(name, false)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return root.Stat(rel)
}

func (me *Component) Exists(name string) (bool, error) {
	_, err := me.stat(name)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

func (me *Component) FileExists(name string) (bool, error) {
	info, err := me.stat(name)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if info.IsDir() {
		return false, fmt.Errorf("`%v` is a directory", name)
	}
	return true, nil
}

func (me *Component) DirExists(name string) (bool, error) {
	info, err := me.stat(name)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.IsDir() {
		return false, fmt.Errorf("`%v` is not a directory", name)
	}
	return true, nil
}

// CreateFile 创建文件；未启用覆盖时由文件系统原子地检查名称是否已存在。
func (me *Component) CreateFile(name string, fileData []byte, options ...storage.Option) error {
	opts := &storage.Options{}
	for _, opt := range options {
		opt(opts)
	}
	root, rel, err := me.openRoot(name, opts.AutoCreateDir)
	if err != nil {
		return err
	}
	defer root.Close()
	if opts.AutoCreateDir {
		if err := root.MkdirAll(filepath.Dir(rel), 0755); err != nil {
			return err
		}
	}
	flag := os.O_CREATE | os.O_WRONLY | os.O_EXCL
	if opts.ForceReplace {
		flag = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	}
	file, err := root.OpenFile(rel, flag, 0644)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(fileData)
	return errors.Join(writeErr, file.Close())
}

func (me *Component) GetFileInfo(name string) (*storage.FileInfo, error) {
	info, err := me.stat(name)
	if err != nil {
		return nil, err
	}
	fullName, err := me.GetFullName(name)
	if err != nil {
		return nil, err
	}
	return &storage.FileInfo{
		IsDir: info.IsDir(), Path: fullName,
		FileSize: info.Size(), UpdateTime: info.ModTime().Unix(),
	}, nil
}

func (me *Component) GetFileData(name string) ([]byte, error) {
	root, rel, err := me.openRoot(name, false)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return root.ReadFile(rel)
}

func (me *Component) GetFullName(name string) (string, error) {
	rootDir, err := filepath.Abs(filepath.Clean(me.config.RootDir))
	if err != nil {
		return "", err
	}
	absName, err := filepath.Abs(filepath.Join(rootDir, name))
	if err != nil {
		return "", err
	}
	rootPrefix := rootDir
	if !strings.HasSuffix(rootPrefix, string(os.PathSeparator)) {
		rootPrefix += string(os.PathSeparator)
	}
	if absName != rootDir && !strings.HasPrefix(absName, rootPrefix) {
		return "", fmt.Errorf("invalid path: %v", name)
	}
	return absName, nil
}
