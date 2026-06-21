package storage

// 本文件属于存储抽象层，统一本地/内存文件系统、复制、监听、MIME、Excel 与图片辅助能力。

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"os"
	"sync"

	"github.com/disintegration/imaging"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/afero"
	"github.com/xuri/excelize/v2"
)

// impl 是 Storage 接口的具体实现
type impl struct {
	config  *Config
	mu      sync.RWMutex
	fs      afero.Fs
	watcher *fsnotify.Watcher
	watches map[string]*watchEntry // 路径 -> 监听条目
	closed  bool
}

// watchEntry 监听条目
type watchEntry struct {
	path    string
	handler WatchHandler
	cancel  context.CancelFunc
}

// New 创建新的 Storage 实例
func New(cfg *Config) (Storage, error) {
	if cfg == nil {
		cfg = &Config{}
		cfg.DefaultConfig()
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	i := &impl{
		config:  cfg,
		watches: make(map[string]*watchEntry),
	}

	// 初始化文件系统
	if err := i.initFileSystem(); err != nil {
		return nil, err
	}

	// 初始化文件监听器
	if cfg.Local.EnableWatch {
		if err := i.initWatcher(); err != nil {
			return nil, err
		}
	}

	return i, nil
}

// initFileSystem 初始化文件系统
func (i *impl) initFileSystem() error {
	localCfg := i.config.normalizedLocal()
	switch localCfg.FSType {
	case FSTypeOS:
		i.fs = afero.NewOsFs()
	case FSTypeMemory:
		i.fs = afero.NewMemMapFs()
	case FSTypeReadOnly:
		i.fs = afero.NewReadOnlyFs(afero.NewOsFs())
	case FSTypeBasePathFS:
		i.fs = afero.NewBasePathFs(afero.NewOsFs(), localCfg.BasePath)
	default:
		return fmt.Errorf("%w: %s", ErrInvalidFSType, localCfg.FSType)
	}
	return nil
}

// initWatcher 初始化文件监听器
func (i *impl) initWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("Storage: failed to create watcher: %w", err)
	}
	i.watcher = watcher
	return nil
}

// FileSystem 返回底层文件系统
func (i *impl) FileSystem() afero.Fs {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.fs
}

// ReadFile 读取文件内容
func (i *impl) ReadFile(path string) ([]byte, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return afero.ReadFile(i.fs, path)
}

// WriteFile 写入文件内容
func (i *impl) WriteFile(path string, data []byte, perm os.FileMode) error {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return afero.WriteFile(i.fs, path, data, perm)
}

// Remove 删除文件或空目录
func (i *impl) Remove(path string) error {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return i.fs.Remove(path)
}

// RemoveAll 递归删除目录
func (i *impl) RemoveAll(path string) error {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return i.fs.RemoveAll(path)
}

// Exists 检查路径是否存在
func (i *impl) Exists(path string) (bool, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return afero.Exists(i.fs, path)
}

// MkdirAll 递归创建目录
func (i *impl) MkdirAll(path string, perm os.FileMode) error {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return i.fs.MkdirAll(path, perm)
}

// IsDir 判断是否为目录
func (i *impl) IsDir(path string) (bool, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return afero.IsDir(i.fs, path)
}

// IsFile 判断是否为文件
func (i *impl) IsFile(path string) (bool, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	isDir, err := afero.IsDir(i.fs, path)
	if err != nil {
		return false, err
	}
	return !isDir, nil
}

// FileSize 获取文件大小
func (i *impl) FileSize(path string) (int64, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	info, err := i.fs.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// ListDir 列出目录内容
func (i *impl) ListDir(path string) ([]os.FileInfo, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return afero.ReadDir(i.fs, path)
}

// OpenExcel 打开 Excel 文件
func (i *impl) OpenExcel(path string) (*excelize.File, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	// 读取文件内容
	data, err := afero.ReadFile(i.fs, path)
	if err != nil {
		return nil, fmt.Errorf("Storage: failed to read excel file: %w", err)
	}

	// 从字节流打开
	file, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("Storage: failed to parse excel file: %w", err)
	}

	return file, nil
}

// CreateExcel 创建新的 Excel 文件
func (i *impl) CreateExcel() *excelize.File {
	return excelize.NewFile()
}

// SaveExcel 保存 Excel 文件
func (i *impl) SaveExcel(file *excelize.File, path string) error {
	i.mu.RLock()
	defer i.mu.RUnlock()

	// 保存到缓冲区
	buf, err := file.WriteToBuffer()
	if err != nil {
		return fmt.Errorf("Storage: failed to write excel to buffer: %w", err)
	}

	// 写入文件系统
	if err := afero.WriteFile(i.fs, path, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("Storage: failed to save excel file: %w", err)
	}

	return nil
}

// ReadExcelSheet 读取 Excel 工作表数据
func (i *impl) ReadExcelSheet(path, sheet string) ([][]string, error) {
	file, err := i.OpenExcel(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	rows, err := file.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("Storage: failed to read sheet %s: %w", sheet, err)
	}

	return rows, nil
}

// OpenImage 打开图片文件
func (i *impl) OpenImage(path string) (image.Image, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	// 读取文件内容
	data, err := afero.ReadFile(i.fs, path)
	if err != nil {
		return nil, fmt.Errorf("Storage: failed to read image file: %w", err)
	}

	// 解码图片
	img, err := imaging.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("Storage: failed to decode image: %w", err)
	}

	return img, nil
}

// SaveImage 保存图片文件
func (i *impl) SaveImage(img image.Image, path string, format imaging.Format) error {
	i.mu.RLock()
	defer i.mu.RUnlock()

	// 编码图片到缓冲区
	var buf bytes.Buffer
	if err := imaging.Encode(&buf, img, format); err != nil {
		return fmt.Errorf("Storage: failed to encode image: %w", err)
	}

	// 写入文件系统
	if err := afero.WriteFile(i.fs, path, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("Storage: failed to save image file: %w", err)
	}

	return nil
}

// ResizeImage 调整图片大小
func (i *impl) ResizeImage(src, dst string, width, height int, format imaging.Format) error {
	// 打开图片
	img, err := i.OpenImage(src)
	if err != nil {
		return err
	}

	// 调整大小
	resized := imaging.Resize(img, width, height, imaging.Lanczos)

	// 保存图片
	return i.SaveImage(resized, dst, format)
}

// CropImage 裁剪图片
func (i *impl) CropImage(src, dst string, rect image.Rectangle, format imaging.Format) error {
	// 打开图片
	img, err := i.OpenImage(src)
	if err != nil {
		return err
	}

	// 裁剪
	cropped := imaging.Crop(img, rect)

	// 保存图片
	return i.SaveImage(cropped, dst, format)
}

// Close 关闭文件服务
func (i *impl) Close() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.closed {
		return nil
	}

	// 停止所有监听
	if i.watcher != nil {
		for path := range i.watches {
			i.watcher.Remove(path)
		}
		i.watcher.Close()
	}

	i.closed = true
	return nil
}

// Reload 重新加载配置
func (i *impl) Reload(ctx context.Context, config *Config) error {
	if err := config.Validate(); err != nil {
		return err
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	// 重新初始化文件系统
	oldFS := i.fs
	oldConfig := i.config
	i.config = config

	if err := i.initFileSystem(); err != nil {
		// 恢复旧配置
		i.config = oldConfig
		i.fs = oldFS
		return err
	}

	return nil
}
