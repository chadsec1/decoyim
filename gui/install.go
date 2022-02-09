package gui

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/coyim/coyim/config"
	"github.com/kardianos/osext"
)

func ensureInstalled() {
	ensureIconsInstalled()
	ensureDesktopFileInstalled()
}

func iconFileName() string {
	return "coyim-" + coyimIcon.hash() + ".png"
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func ensureIconsInstalled() {
	dataHome := config.SystemDataDir()
	if dataHome != "" && fileExists(dataHome) {
		icon16, _ := coyimIcon.createPixBufWithSize(16, 16)
		icon32, _ := coyimIcon.createPixBufWithSize(32, 32)
		icon48, _ := coyimIcon.createPixBufWithSize(48, 48)
		icon128, _ := coyimIcon.createPixBufWithSize(128, 128)
		icon256, _ := coyimIcon.createPixBufWithSize(256, 256)

		_ = os.MkdirAll(filepath.Join(dataHome, "icons/hicolor/16x16/apps"), 0700)
		_ = os.MkdirAll(filepath.Join(dataHome, "icons/hicolor/32x32/apps"), 0700)
		_ = os.MkdirAll(filepath.Join(dataHome, "icons/hicolor/48x48/apps"), 0700)
		_ = os.MkdirAll(filepath.Join(dataHome, "icons/hicolor/128x128/apps"), 0700)
		_ = os.MkdirAll(filepath.Join(dataHome, "icons/hicolor/256x256/apps"), 0700)

		fileName := iconFileName()
		_ = icon16.SavePNG(filepath.Join(dataHome, "icons/hicolor/16x16/apps", fileName), 9)
		_ = icon32.SavePNG(filepath.Join(dataHome, "icons/hicolor/32x32/apps", fileName), 9)
		_ = icon48.SavePNG(filepath.Join(dataHome, "icons/hicolor/48x48/apps", fileName), 9)
		_ = icon128.SavePNG(filepath.Join(dataHome, "icons/hicolor/128x128/apps", fileName), 9)
		_ = icon256.SavePNG(filepath.Join(dataHome, "icons/hicolor/256x256/apps", fileName), 9)
	}
}

func ensureDesktopFileInstalled() {
	dataHome := config.SystemDataDir()
	if dataHome != "" && fileExists(dataHome) {
		_ = os.MkdirAll(filepath.Join(dataHome, "applications"), 0700)
		_ = ioutil.WriteFile(filepath.Join(dataHome, "applications", "coyim.desktop"), []byte(generateDesktopFile(false)), 0600)
		_ = ioutil.WriteFile(filepath.Join(dataHome, "applications", "coyim-debug.desktop"), []byte(generateDesktopFile(true)), 0600)
	}
}

func generateDesktopFile(debug bool) string {
	name := "CoyIM"
	path, _ := osext.Executable()
	icon := iconFileName()
	revision := coyimVersion

	if debug {
		name = name + "-debug"
		path = path + " -debug"
	}

	return "" +
		"[Desktop Entry]\n" +
		"Type=Application\n" +
		"Version=1.0\n" +
		"Encoding=UTF-8\n" +
		"Name=" + name + "\n" +
		"Comment=Secure Instant Messenger\n" +
		"Exec=" + path + "\n" +
		"Icon=" + icon + "\n" +
		"Terminal=false\n" +
		"Categories=Network;\n" +
		"# CoyIMVersion=" + revision + "\n"
}
