//go:build darwin

package wechat

import (
	"os"
	"path/filepath"
)

type macosPlatform struct {
}

func newPlatform() platform {
	return &macosPlatform{}
}

func (m *macosPlatform) GetDefaultPaths(log ...LogFunc) []string {
	var result []string
	l := getLogger(log)

	l("[macOS] 开始获取 WeChat 默认路径")

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		l("[macOS] 获取用户目录失败: %v", err)
		return result
	}
	l("[macOS] 用户目录: %s", userHomeDir)

	// v3
	v3Path := filepath.Join(userHomeDir, "/Library/Containers/com.tencent.xinWeChat/Data/.wxapplet/packages")
	l("  [v3] 检测路径: %s", v3Path)
	if fileInfo, err := os.Stat(v3Path); err != nil {
		l("  [v3] 路径不存在: %v", err)
	} else if !fileInfo.IsDir() {
		l("  [v3] 存在但不是目录，跳过")
	} else {
		l("  [v3] 目录存在，加入结果")
		result = append(result, v3Path)
	}

	// v4
	v4Path := filepath.Join(userHomeDir, "/Library/Containers/com.tencent.xinWeChat/Data/Documents/app_data/radium/Applet/packages")
	l("  [v4] 检测路径: %s", v4Path)
	if fileInfo, err := os.Stat(v4Path); err != nil {
		l("  [v4] 路径不存在: %v", err)
	} else if !fileInfo.IsDir() {
		l("  [v4] 存在但不是目录，跳过")
	} else {
		l("  [v4] 目录存在，加入结果")
		result = append(result, v4Path)
	}

	l("[macOS] 获取完成，共 %d 个路径", len(result))
	return result
}
