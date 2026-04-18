package wechat

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildIconIndexFiltersAndDeduplicates(t *testing.T) {
	iconCache.reset()

	rootDir := t.TempDir()
	scanDir := filepath.Join(rootDir, "install-root")
	packagesDir := filepath.Join(rootDir, "packages")

	if err := os.MkdirAll(scanDir, 0o755); err != nil {
		t.Fatalf("create scan dir: %v", err)
	}
	if err := os.MkdirAll(packagesDir, 0o755); err != nil {
		t.Fatalf("create packages dir: %v", err)
	}

	validFirst := filepath.Join(packagesDir, "wxabc123_alpha.png")
	validSecond := filepath.Join(packagesDir, "wxabc123_zeta.png")
	validOther := filepath.Join(packagesDir, "wxdef456_cover.png")
	ignoredFiles := []string{
		filepath.Join(packagesDir, "wxABC123_uppercase.png"),
		filepath.Join(packagesDir, "wxabc123.png"),
		filepath.Join(packagesDir, "random.png"),
		filepath.Join(packagesDir, "wxabc123_notes.txt"),
	}

	for _, file := range append([]string{validFirst, validSecond, validOther}, ignoredFiles...) {
		if err := os.WriteFile(file, []byte(filepath.Base(file)), 0o644); err != nil {
			t.Fatalf("write %s: %v", file, err)
		}
	}
	if err := os.Mkdir(filepath.Join(packagesDir, "wxdir123_extra.png"), 0o755); err != nil {
		t.Fatalf("create ignored dir: %v", err)
	}

	index, err := buildIconIndex(scanDir)
	if err != nil {
		t.Fatalf("buildIconIndex returned error: %v", err)
	}

	if len(index) != 2 {
		t.Fatalf("expected 2 icon entries, got %d", len(index))
	}
	if got := index["wxabc123"]; got != validFirst {
		t.Fatalf("expected duplicate wxid to keep first file %q, got %q", validFirst, got)
	}
	if got := index["wxdef456"]; got != validOther {
		t.Fatalf("expected wxdef456 to map to %q, got %q", validOther, got)
	}
	if _, exists := index["wxABC123"]; exists {
		t.Fatalf("expected invalid uppercase wxid to be ignored")
	}
}

func TestScanWxapkgItemAddsIconFieldsForScannedDirectories(t *testing.T) {
	iconCache.reset()

	rootDir := t.TempDir()
	packagesDir := filepath.Join(rootDir, "packages")
	iconDir := filepath.Join(rootDir, "icon")
	wxid := "wxabcdef1234567890"
	appDir := filepath.Join(packagesDir, wxid)
	iconBytes := []byte("png-bytes")
	iconPath := filepath.Join(iconDir, wxid+"_home.png")

	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("create app dir: %v", err)
	}
	if err := os.MkdirAll(iconDir, 0o755); err != nil {
		t.Fatalf("create icon dir: %v", err)
	}
	if err := os.WriteFile(iconPath, iconBytes, 0o644); err != nil {
		t.Fatalf("write icon file: %v", err)
	}

	items, err := ScanWxapkgItem(packagesDir, true)
	if err != nil {
		t.Fatalf("ScanWxapkgItem returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	expectedDataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(iconBytes)
	if items[0].IconPath != iconPath {
		t.Fatalf("expected icon path %q, got %q", iconPath, items[0].IconPath)
	}
	if items[0].IconDataURL != expectedDataURL {
		t.Fatalf("expected icon data URL %q, got %q", expectedDataURL, items[0].IconDataURL)
	}
}

func TestScanWxapkgItemLeavesIconEmptyForManualMode(t *testing.T) {
	iconCache.reset()

	rootDir := t.TempDir()
	wxid := "wxabcdef1234567890"
	packagesDir := filepath.Join(rootDir, "packages")
	appDir := filepath.Join(packagesDir, wxid)
	iconPath := filepath.Join(packagesDir, wxid+"_home.png")

	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("create app dir: %v", err)
	}
	if err := os.WriteFile(iconPath, []byte("png-bytes"), 0o644); err != nil {
		t.Fatalf("write icon file: %v", err)
	}

	items, err := ScanWxapkgItem(appDir, false)
	if err != nil {
		t.Fatalf("ScanWxapkgItem returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].IconPath != "" {
		t.Fatalf("expected empty icon path in manual mode, got %q", items[0].IconPath)
	}
	if items[0].IconDataURL != "" {
		t.Fatalf("expected empty icon data URL in manual mode, got %q", items[0].IconDataURL)
	}
}

func TestScanWxapkgItemUsesWarmCacheBeforeSearchingLocalPackages(t *testing.T) {
	iconCache.reset()

	wxid := "wxabcdef1234567890"
	iconBytes := []byte("cached-png")

	seedRoot := t.TempDir()
	seedPackagesDir := filepath.Join(seedRoot, "packages")
	seedIconDir := filepath.Join(seedRoot, "icon")
	if err := os.MkdirAll(filepath.Join(seedPackagesDir, wxid), 0o755); err != nil {
		t.Fatalf("create seed app dir: %v", err)
	}
	if err := os.MkdirAll(seedIconDir, 0o755); err != nil {
		t.Fatalf("create seed icon dir: %v", err)
	}
	seedIconPath := filepath.Join(seedIconDir, wxid+"_seed.png")
	if err := os.WriteFile(seedIconPath, iconBytes, 0o644); err != nil {
		t.Fatalf("write seed icon file: %v", err)
	}

	WarmIconCache([]string{seedPackagesDir})

	scanRoot := t.TempDir()
	scanPackagesDir := filepath.Join(scanRoot, "packages")
	if err := os.MkdirAll(filepath.Join(scanPackagesDir, wxid), 0o755); err != nil {
		t.Fatalf("create scan app dir: %v", err)
	}

	items, err := ScanWxapkgItem(scanPackagesDir, true)
	if err != nil {
		t.Fatalf("ScanWxapkgItem returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	expectedDataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(iconBytes)
	if items[0].IconPath != seedIconPath {
		t.Fatalf("expected cached icon path %q, got %q", seedIconPath, items[0].IconPath)
	}
	if items[0].IconDataURL != expectedDataURL {
		t.Fatalf("expected cached icon data URL %q, got %q", expectedDataURL, items[0].IconDataURL)
	}
}
