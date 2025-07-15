// +build !windows

package importer

import (
	"path/filepath"

	"github.com/chadsec1/decoyim/config"
)

func gajimGetConfigAndDataDirs() (configRoot, dataRoot string) {
	configRoot = filepath.Join(config.SystemConfigDir(), "gajim")
	dataRoot = filepath.Join(config.SystemDataDir(), "gajim")
	return
}
