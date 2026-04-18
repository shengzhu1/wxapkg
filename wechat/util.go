package wechat

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/base64"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	uuidlib "github.com/google/uuid"
	"golang.org/x/crypto/pbkdf2"
)

var reWxIconFile = regexp.MustCompile(`^(wx[a-z0-9]+)_.+\.png$`)

type iconCacheEntry struct {
	Path    string
	DataURL string
}

type wxIconCache struct {
	locker          sync.RWMutex
	icons           map[string]iconCacheEntry
	scannedPackages map[string]bool
}

func newWxIconCache() *wxIconCache {
	return &wxIconCache{
		icons:           make(map[string]iconCacheEntry),
		scannedPackages: make(map[string]bool),
	}
}

func (c *wxIconCache) reset() {
	c.locker.Lock()
	defer c.locker.Unlock()

	c.icons = make(map[string]iconCacheEntry)
	c.scannedPackages = make(map[string]bool)
}

func (c *wxIconCache) get(wxid string) (iconCacheEntry, bool) {
	c.locker.RLock()
	defer c.locker.RUnlock()

	entry, ok := c.icons[wxid]
	return entry, ok
}

func (c *wxIconCache) setIfAbsent(wxid string, entry iconCacheEntry) {
	c.locker.Lock()
	defer c.locker.Unlock()

	if _, exists := c.icons[wxid]; exists {
		return
	}
	c.icons[wxid] = entry
}

func (c *wxIconCache) hasScanned(packagesDir string) bool {
	c.locker.RLock()
	defer c.locker.RUnlock()

	return c.scannedPackages[packagesDir]
}

func (c *wxIconCache) markScanned(packagesDir string) {
	c.locker.Lock()
	defer c.locker.Unlock()

	c.scannedPackages[packagesDir] = true
}

var iconCache = newWxIconCache()

func getPathSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	if !info.IsDir() {
		return info.Size()
	}
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

func uuid() string {
	newUUID, _ := uuidlib.NewRandom()
	return newUUID.String()
}

// ListFilesWithExtension extension eg: .wxapkg
func ListFilesWithExtension(dir, extension string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, extension) {
			paths = append(paths, path)
		}
		return nil
	})

	return paths, err
}

func decryptWxapkgFile(wxid string, dataByte []byte) ([]byte, error) {
	var (
		salt = "saltiest"
		iv   = "the iv: 16 bytes"
	)

	dk := pbkdf2.Key([]byte(wxid), []byte(salt), 1000, 32, sha1.New)
	block, _ := aes.NewCipher(dk)
	blockMode := cipher.NewCBCDecrypter(block, []byte(iv))
	originData := make([]byte, 1024)
	blockMode.CryptBlocks(originData, dataByte[6:1024+6])

	afData := make([]byte, len(dataByte)-1024-6) // remove first 6 + 1024 byte
	var xorKey = byte(0x66)
	if len(wxid) >= 2 {
		xorKey = wxid[len(wxid)-2]
	}
	for i, b := range dataByte[1024+6:] { // from 6 + 1024 byte
		afData[i] = b ^ xorKey
	}

	originData = append(originData[:1023], afData...)

	return originData, nil
}

func resolveIconSearchDirs(path string) []string {
	base := filepath.Base(path)
	parent := filepath.Dir(path)

	switch base {
	case "icon":
		return []string{path, filepath.Join(parent, "packages")}
	case "packages":
		return []string{filepath.Join(parent, "icon"), path}
	default:
		return []string{filepath.Join(parent, "icon"), filepath.Join(parent, "packages")}
	}
}

func buildIconIndex(scanPath string) (map[string]string, error) {
	index := make(map[string]string)
	for _, searchDir := range resolveIconSearchDirs(scanPath) {
		entries, err := os.ReadDir(searchDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return map[string]string{}, nil
		}

		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Name() < entries[j].Name()
		})

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			matches := reWxIconFile.FindStringSubmatch(entry.Name())
			if len(matches) != 2 {
				continue
			}

			wxid := matches[1]
			if _, exists := index[wxid]; exists {
				continue
			}
			index[wxid] = filepath.Join(searchDir, entry.Name())
		}
	}

	return index, nil
}

func buildIconDataURL(iconPath string) (string, error) {
	iconData, err := os.ReadFile(iconPath)
	if err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(iconData), nil
}

func scanIconsIntoCache(scanPath string) error {
	for _, searchDir := range resolveIconSearchDirs(scanPath) {
		if iconCache.hasScanned(searchDir) {
			continue
		}

		info, err := os.Stat(searchDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if !info.IsDir() {
			continue
		}

		entries, err := os.ReadDir(searchDir)
		if err != nil {
			return err
		}
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Name() < entries[j].Name()
		})

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			matches := reWxIconFile.FindStringSubmatch(entry.Name())
			if len(matches) != 2 {
				continue
			}

			wxid := matches[1]
			if _, exists := iconCache.get(wxid); exists {
				continue
			}

			iconPath := filepath.Join(searchDir, entry.Name())
			iconDataURL, err := buildIconDataURL(iconPath)
			if err != nil {
				continue
			}
			iconCache.setIfAbsent(wxid, iconCacheEntry{
				Path:    iconPath,
				DataURL: iconDataURL,
			})
		}
		iconCache.markScanned(searchDir)
	}

	return nil
}

func WarmIconCache(paths []string) {
	for _, path := range paths {
		_ = scanIconsIntoCache(path)
	}
}

func enrichWxapkgItemsWithIcons(items []WxapkgItem, scanPath string) error {
	hasMiss := false
	for i := range items {
		entry, exists := iconCache.get(items[i].WxId)
		if !exists {
			hasMiss = true
			continue
		}
		items[i].IconPath = entry.Path
		items[i].IconDataURL = entry.DataURL
	}

	if !hasMiss {
		return nil
	}

	if err := scanIconsIntoCache(scanPath); err != nil {
		return err
	}

	for i := range items {
		if items[i].IconDataURL != "" {
			continue
		}

		entry, exists := iconCache.get(items[i].WxId)
		if !exists {
			continue
		}
		items[i].IconPath = entry.Path
		items[i].IconDataURL = entry.DataURL
	}

	return nil
}

func ScanWxapkgItem(path string, scan bool) ([]WxapkgItem, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if !stat.IsDir() || !scan {
		return []WxapkgItem{{
			UUID:           uuid(),
			WxId:           "unknown",
			Location:       path,
			Size:           getPathSize(path),
			IsDir:          stat.IsDir(),
			LastModifyTime: stat.ModTime().Unix(),
		}}, nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var result []WxapkgItem

	for _, entry := range entries {
		if entry.IsDir() {
			dirName := entry.Name()
			if reWxId.MatchString(dirName) {
				localPath := filepath.Join(path, dirName)

				item := WxapkgItem{
					UUID:       uuid(),
					WxId:       dirName,
					Location:   localPath,
					EncryptKey: dirName,
					Size:       getPathSize(localPath),
					IsDir:      true,
				}
				if info, err := os.Stat(localPath); err == nil {
					item.LastModifyTime = info.ModTime().Unix()
				}

				result = append(result, item)
			}
		}
	}

	if err := enrichWxapkgItemsWithIcons(result, path); err != nil {
		return result, nil
	}

	return result, nil
}
