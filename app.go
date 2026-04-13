package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wux1an/wxapkg/wechat"
)

const EventUnpackProgress = "unpack:progress-changed"

type AppService struct {
	ctx   *application.App
	items sync.Map // map[string]*wechat.WxapkgItem
}

func NewAppService() *AppService {
	return &AppService{}
}

func (a *AppService) SetContext(app *application.App) {
	a.ctx = app
}

// GetWxapkgItem returns the latest state of the item by UUID (push-pull pattern)
func (a *AppService) GetWxapkgItem(uuid string) *wechat.WxapkgItem {
	if val, ok := a.items.Load(uuid); ok {
		return val.(*wechat.WxapkgItem)
	}
	return nil
}

// storeItem stores the item and returns a pointer to it for modification
func (a *AppService) storeItem(item *wechat.WxapkgItem) *wechat.WxapkgItem {
	a.items.Store(item.UUID, item)
	return item
}

func (a *AppService) OpenDirectoryDialog(title string, root string) (string, error) {
	if _, err := os.Stat(root); os.IsNotExist(err) {
		root = "" // if directory not exist, will panic
	}
	return a.ctx.Dialog.OpenFile().
		SetTitle(title).
		SetDirectory(root).
		CanChooseDirectories(true).
		CanChooseFiles(false).
		PromptForSingleSelection()
}

func (a *AppService) OpenFileDialog(title string, root string, filters []application.FileFilter) (string, error) {
	if _, err := os.Stat(root); os.IsNotExist(err) {
		root = "" // if directory not exist, will panic
	}
	var dialog = a.ctx.Dialog.OpenFile().
		SetTitle(title).
		SetDirectory(root).
		CanChooseDirectories(false).
		CanChooseFiles(true)

	if len(filters) > 0 {
		for _, filter := range filters {
			dialog.AddFilter(filter.DisplayName, filter.Pattern)
		}
	}

	return dialog.PromptForSingleSelection()
}

func (a *AppService) GetDefaultPaths() []string {
	return wechat.Platform.GetDefaultPaths()
}

func (a *AppService) ScanWxapkgItem(path string, scan bool) ([]wechat.WxapkgItem, error) {
	return wechat.ScanWxapkgItem(path, scan)
}

func (a *AppService) ClipboardSetText(text string) bool {
	return a.ctx.Clipboard.SetText(text)
}

func (a *AppService) UnpackWxapkgItem(item wechat.WxapkgItem, options wechat.UnpackOptions) {
	itemPtr := a.storeItem(&item)
	wechat.NewUnpacker(itemPtr, &options).UnpackWithStatusCallback(func(item *wechat.WxapkgItem) {
		// Push-pull pattern: only emit UUID, frontend pulls latest state via GetWxapkgItem
		a.ctx.Event.Emit(EventUnpackProgress, item.UUID)
	})
}

func (a *AppService) Version() string {
	return version
}

func (a *AppService) Github() string {
	return github
}

func (a *AppService) OpenUrl(url string) error {
	return a.ctx.Browser.OpenURL(url)
}

func (a *AppService) OpenPath(path string) error {
	if path == "" {
		return errors.New("this path is empty")
	}

	var opener = map[string]string{
		"windows": "explorer",
		"darwin":  "open",
		"linux":   "xdg-open",
	}

	cmd, ok := opener[runtime.GOOS]
	if !ok {
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	return exec.Command(cmd, path).Start()
}

// ComputeSavePath 计算输出路径，根据平台自动使用正确的路径分隔符
func (a *AppService) ComputeSavePath(outputDir string, wxapkgPath string) string {
	absDir, err := filepath.Abs(outputDir)
	if err != nil {
		return outputDir
	}
	// 提取文件名（不含扩展名）
	baseName := filepath.Base(wxapkgPath)
	ext := filepath.Ext(baseName)
	if ext != "" {
		baseName = baseName[:len(baseName)-len(ext)]
	}
	return filepath.Join(absDir, baseName)
}
