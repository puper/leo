package localfile

import (
	"fmt"
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

// CreateDir 创建目录
func (me *Component) CreateDir(name string) error {
	name, err := me.GetFullName(name)
	if err != nil {
		return err
	}
	return os.MkdirAll(name, 0755)
}

// RemoveDir 删除目录
func (me *Component) Remove(name string, options ...storage.Option) error {
	name, err := me.GetFullName(name)
	if err != nil {
		return err
	}
	opts := &storage.Options{}
	if len(options) > 0 {
		for _, opt := range options {
			opt(opts)
		}
	}
	if opts.RemoveAll {
		return os.RemoveAll(name)
	}
	return os.Remove(name)
}

// ListDir 列出目录中的文件和子目录
func (me *Component) ListDir(name string) ([]*storage.FileInfo, error) {
	name, err := me.GetFullName(name)
	if err != nil {
		return nil, err
	}
	files, err := os.ReadDir(name)
	if err != nil {
		return nil, err
	}

	var fileInfos []*storage.FileInfo
	for _, file := range files {
		if file.IsDir() {
			fileInfos = append(fileInfos, &storage.FileInfo{
				IsDir: true,
				Path:  filepath.Join(name, file.Name()),
			})
			continue
		}
		t, err := file.Info()
		if err == nil {
			info := &storage.FileInfo{
				Path:       filepath.Join(name, file.Name()),
				FileSize:   t.Size(),
				UpdateTime: t.ModTime().Unix(),
			}
			fileInfos = append(fileInfos, info)
		}
	}
	return fileInfos, nil
}

func (me *Component) Exists(name string) (bool, error) {
	name, err := me.GetFullName(name)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(name)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (me *Component) FileExists(name string) (bool, error) {
	name, err := me.GetFullName(name)
	if err != nil {
		return false, err
	}
	fileInfo, err := os.Stat(name)
	if err == nil {
		if fileInfo.IsDir() {
			return false, fmt.Errorf("`%v` is a directory", name)
		}
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (me *Component) DirExists(name string) (bool, error) {
	name, err := me.GetFullName(name)
	if err != nil {
		return false, err
	}
	fileInfo, err := os.Stat(name)
	if err == nil {
		if fileInfo.IsDir() {
			return true, nil
		}
		return false, fmt.Errorf("`%v` is not a directory", name)
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// CreateFile 创建文件
func (me *Component) CreateFile(name string, fileData []byte, options ...storage.Option) error {
	name, err := me.GetFullName(name)
	if err != nil {
		return err
	}
	opts := &storage.Options{}
	if len(options) > 0 {
		for _, opt := range options {
			opt(opts)
		}
	}
	if !opts.ForceReplace {
		_, err := os.Stat(name)
		if err == nil {
			return fmt.Errorf("file `%v` exists", name)
		}
		if !os.IsNotExist(err) {
			return err
		}
	}
	if opts.AutoCreateDir {
		os.MkdirAll(filepath.Dir(name), 0755)
	}
	return os.WriteFile(name, fileData, 0644)
}

// GetFileInfo 获取文件信息
func (me *Component) GetFileInfo(name string) (*storage.FileInfo, error) {
	name, err := me.GetFullName(name)
	if err != nil {
		return nil, err
	}
	fileInfo, err := os.Stat(name)
	if err != nil {
		return nil, err
	}
	return &storage.FileInfo{
		Path:       name,
		FileSize:   fileInfo.Size(),
		UpdateTime: fileInfo.ModTime().Unix(),
	}, nil
}

// GetFileData 获取文件数据
func (me *Component) GetFileData(name string) ([]byte, error) {
	if exists, err := me.FileExists(name); !exists {
		return nil, err
	}
	name, err := me.GetFullName(name)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(name)
}

func (me *Component) GetFullName(name string) (string, error) {
	name = filepath.Join(me.config.RootDir, name)
	absName, err := filepath.Abs(name)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(absName, name) {
		return "", fmt.Errorf("invalid path: %v", name)
	}
	return absName, nil
}
