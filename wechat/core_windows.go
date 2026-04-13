//go:build windows

package wechat

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

type windowsPlatform struct{}

func newPlatform() platform { return &windowsPlatform{} }

func (m *windowsPlatform) GetDefaultPaths(log ...LogFunc) []string {
	var paths []string
	l := getLogger(log)

	l("[Windows] 开始获取 WeChat 默认路径")

	v3 := m.getPathOfV3(l)
	if v3 != "" {
		l("  [v3] 路径有效，加入结果: %s", v3)
		paths = append(paths, v3)
	}

	v4Root := m.getPathOfV4Root(l)
	if v4Root != "" {
		l("  [v4] 路径有效，加入结果: %s", v4Root)
		paths = append(paths, v4Root)
	}

	// v4 users 目录：radium/users/<user_id>/applet/packages
	v4Users := m.getPathOfV4Users(l)
	if len(v4Users) > 0 {
		for _, p := range v4Users {
			l("  [v4-users] 路径有效，加入结果: %s", p)
			paths = append(paths, p)
		}
	}

	l("[Windows] 获取完成，共 %d 个路径", len(paths))
	return paths
}

func (m *windowsPlatform) getPathOfV4Root(l LogFunc) string {
	l("  [v4] 开始检测 (直接路径)...")
	appDataDir, err := os.UserConfigDir()
	if err != nil {
		l("  [v4] 获取 UserConfigDir 失败: %v", err)
		return ""
	}
	l("  [v4] UserConfigDir = %s", appDataDir)

	v4Path := filepath.Join(appDataDir, "Tencent", "xwechat", "radium", "Applet", "packages")
	l("  [v4] 拼接路径: %s", v4Path)

	if fileInfo, err := os.Stat(v4Path); err != nil {
		l("  [v4] 路径不存在: %v", err)
		return ""
	} else if !fileInfo.IsDir() {
		l("  [v4] 存在但不是目录，跳过")
		return ""
	}
	l("  [v4] 目录存在，有效")
	return v4Path
}

func (m *windowsPlatform) getPathOfV4Users(l LogFunc) []string {
	l("  [v4-users] 开始检测 (users 子目录)...")
	appDataDir, err := os.UserConfigDir()
	if err != nil {
		l("  [v4-users] 获取 UserConfigDir 失败: %v", err)
		return nil
	}

	usersDir := filepath.Join(appDataDir, "Tencent", "xwechat", "radium", "users")
	l("  [v4-users] 检测 users 目录: %s", usersDir)

	entries, err := os.ReadDir(usersDir)
	if err != nil {
		l("  [v4-users] 读取 users 目录失败: %v", err)
		return nil
	}

	var result []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		userDir := filepath.Join(usersDir, entry.Name(), "applet", "packages")
		l("  [v4-users] 检测用户目录: %s", userDir)
		if fileInfo, err := os.Stat(userDir); err != nil {
			l("  [v4-users]   %s 不存在: %v", entry.Name(), err)
		} else if !fileInfo.IsDir() {
			l("  [v4-users]   %s 不是目录，跳过", entry.Name())
		} else {
			l("  [v4-users]   %s 目录存在，有效", entry.Name())
			result = append(result, userDir)
		}
	}

	if len(result) == 0 {
		l("  [v4-users] 未找到任何有效的 users 子目录")
	}
	return result
}

func (m *windowsPlatform) getPathOfV3(l LogFunc) string {
	l("  [v3] 开始检测...")
	l("  [v3] 尝试读取注册表: HKCU\\Software\\Tencent\\WeChat\\FileSavePath")

	wechatKey, err := registry.OpenKey(registry.CURRENT_USER, `Software\Tencent\WeChat`, registry.QUERY_VALUE)
	if err != nil {
		l("  [v3] 打开注册表键失败: %v", err)
		return ""
	}
	defer wechatKey.Close()

	value, _, err := wechatKey.GetStringValue("FileSavePath")
	if err != nil {
		l("  [v3] 读取 FileSavePath 值失败: %v", err)
		return ""
	}
	l("  [v3] 注册表原始值: %q", value)

	if value == "MyDocument:" {
		docPath := os.Getenv("USERPROFILE")
		l("  [v3] 检测到特殊值 'MyDocument:'，拼接用户目录: %s", docPath)
		value = filepath.Join(docPath, "Documents")
	}

	v3Path := filepath.Join(value, "WeChat Files")
	l("  [v3] 拼接最终路径: %s", v3Path)
	return v3Path
}
