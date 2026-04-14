//go:build darwin

package wechat

import (
	"os"
	"path/filepath"
	"strings"
)

type macosPlatform struct{}

func newPlatform() platform { return &macosPlatform{} }

func (m *macosPlatform) GetDefaultPaths() PathScanResult {
	var b strings.Builder
	log := func(line string) { b.WriteString(line + "\n") }

	var result []string

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		log("获取用户目录失败: " + err.Error())
		return PathScanResult{Paths: result, Logs: b.String()}
	}

	// v3
	v3Path := filepath.Join(userHomeDir, "Library/Containers/com.tencent.xinWeChat/Data/.wxapplet/packages")
	log("1. 检测 v3\n   路径: " + v3Path)
	if fileInfo, err := os.Stat(v3Path); err == nil && fileInfo.IsDir() {
		log("   结果: 有效")
		result = append(result, v3Path)
	} else {
		log("   结果: 目录不存在")
	}

	// v4
	v4Path := filepath.Join(userHomeDir, "Library/Containers/com.tencent.xinWeChat/Data/Documents/app_data/radium/Applet/packages")
	log("2. 检测 v4\n   路径: " + v4Path)
	if fileInfo, err := os.Stat(v4Path); err == nil && fileInfo.IsDir() {
		log("   结果: 有效")
		result = append(result, v4Path)
	} else {
		log("   结果: 目录不存在")
	}

	return PathScanResult{Paths: result, Logs: b.String()}
}
