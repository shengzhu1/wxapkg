//go:build windows

package wechat

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

type windowsPlatform struct{}

func newPlatform() platform { return &windowsPlatform{} }

func (m *windowsPlatform) GetDefaultPaths() PathScanResult {
	var b strings.Builder
	log := func(line string) { b.WriteString(line + "\n") }

	var paths []string

	// ── Step 1: v4 根目录 ──
	appDataDir, _ := os.UserConfigDir()
	v4Path := filepath.Join(appDataDir, "Tencent", "xwechat", "radium", "Applet", "packages")
	log(fmt.Sprintf("1. 检测 v4 根目录\n   路径: %s", v4Path))
	if fileInfo, err := os.Stat(v4Path); err == nil && fileInfo.IsDir() {
		log("   结果: 有效")
		paths = append(paths, v4Path)
	} else {
		log("   结果: 目录不存在")
	}

	// ── Step 2: v3 注册表 ──
	log("2. 读取 v3 注册表\n   键: HKCU\\Software\\Tencent\\WeChat\\FileSavePath")
	wechatKey, err := registry.OpenKey(registry.CURRENT_USER, `Software\Tencent\WeChat`, registry.QUERY_VALUE)
	if err != nil {
		log("   结果: 读取失败")
	} else {
		defer wechatKey.Close()
		value, _, err := wechatKey.GetStringValue("FileSavePath")
		if err != nil {
			log("   结果: 未找到该值")
		} else {
			log(fmt.Sprintf("   原始值: %q", value))
			if value == "MyDocument:" {
				value = filepath.Join(os.Getenv("USERPROFILE"), "Documents")
				log(fmt.Sprintf("   转换值: %s", value))
			}
			v3Path := filepath.Join(value, "WeChat Files")
			log(fmt.Sprintf("   最终路径: %s", v3Path))
			if fileInfo, err := os.Stat(v3Path); err == nil && fileInfo.IsDir() {
				log("   结果: 有效")
				paths = append(paths, v3Path)
			} else {
				log("   结果: 目录不存在")
			}
		}
	}

	// ── Step 3: v4 users ──
	usersDir := filepath.Join(appDataDir, "Tencent", "xwechat", "radium", "users")
	log(fmt.Sprintf("3. 扫描 v4 用户目录\n   路径: %s", usersDir))
	entries, err := os.ReadDir(usersDir)
	if err != nil || entries == nil {
		log("   结果: 未找到")
	} else {
		var found []string
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			userDir := filepath.Join(usersDir, entry.Name(), "applet", "packages")
			if fileInfo, err := os.Stat(userDir); err == nil && fileInfo.IsDir() {
				paths = append(paths, userDir)
				found = append(found, entry.Name())
			}
		}
		if len(found) == 0 {
			log("   结果: 未找到有效用户")
		} else {
			log(fmt.Sprintf("   有效用户: %v", found))
		}
	}

	return PathScanResult{Paths: paths, Logs: b.String()}
}
