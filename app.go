package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/wux1an/wxapkg/wechat"
)

type AppService struct {
	ctx   context.Context
	items sync.Map // map[string]*wechat.WxapkgItem
}

func NewAppService() *AppService {
	return &AppService{}
}

func (a *AppService) startup(ctx context.Context) {
	a.ctx = ctx
}

// storeItem stores the item and returns a pointer to it for modification
func (a *AppService) storeItem(item *wechat.WxapkgItem) *wechat.WxapkgItem {
	a.items.Store(item.UUID, item)
	return item
}

// GetWxapkgItem returns the latest state of the item by UUID (push-pull pattern)
func (a *AppService) GetWxapkgItem(uuid string) *wechat.WxapkgItem {
	if val, ok := a.items.Load(uuid); ok {
		return val.(*wechat.WxapkgItem)
	}
	return nil
}

func (a *AppService) OpenDirectoryDialog(title string, root string) (string, error) {
	if _, err := os.Stat(root); os.IsNotExist(err) {
		root = ""
	}
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            title,
		DefaultDirectory: root,
	})
}

func (a *AppService) OpenFileDialog(title string, root string, filters []FileFilter) (string, error) {
	if _, err := os.Stat(root); os.IsNotExist(err) {
		root = ""
	}
	goFilters := make([]runtime.FileFilter, len(filters))
	for i, f := range filters {
		goFilters[i] = runtime.FileFilter{DisplayName: f.DisplayName, Pattern: f.Pattern}
	}
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            title,
		DefaultDirectory: root,
		Filters:          goFilters,
	})
}

func (a *AppService) GetDefaultPaths() wechat.PathScanResult {
	return wechat.Platform.GetDefaultPaths()
}

func (a *AppService) ScanWxapkgItem(path string, scan bool) ([]wechat.WxapkgItem, error) {
	return wechat.ScanWxapkgItem(path, scan)
}

func (a *AppService) ClipboardSetText(text string) error {
	return runtime.ClipboardSetText(a.ctx, text)
}

func (a *AppService) UnpackWxapkgItem(item wechat.WxapkgItem, options wechat.UnpackOptions) {
	itemPtr := a.storeItem(&item)
	ctx := a.ctx
	wechat.NewUnpacker(itemPtr, &options).UnpackWithStatusCallback(func(item *wechat.WxapkgItem) {
		a.items.Store(item.UUID, item)
		runtime.EventsEmit(ctx, "unpack:progress-changed", item.UUID)
	})
}

func (a *AppService) Version() string {
	return version
}

func (a *AppService) Github() string {
	return github
}

func (a *AppService) OpenUrl(url string) error {
	runtime.BrowserOpenURL(a.ctx, url)
	return nil
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
	cmd, ok := opener[goruntime.GOOS]
	if !ok {
		return fmt.Errorf("unsupported OS: %s", goruntime.GOOS)
	}
	return exec.Command(cmd, path).Start()
}

func (a *AppService) ComputeSavePath(outputDir string, wxapkgPath string) string {
	absDir, err := filepath.Abs(outputDir)
	if err != nil {
		return outputDir
	}
	baseName := filepath.Base(wxapkgPath)
	ext := filepath.Ext(baseName)
	if ext != "" {
		baseName = baseName[:len(baseName)-len(ext)]
	}
	return filepath.Join(absDir, baseName+"_unpacked")
}

type FileFilter struct {
	DisplayName string
	Pattern     string
}

func (a *AppService) shutdown() {}

func (a *AppService) beforeClose() {}
