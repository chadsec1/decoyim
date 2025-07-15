package importer

import (
	"os"
	"path/filepath"

	"github.com/chadsec1/decoyim/config"
)

func findDirOSDependent() (string, bool) {
	app := filepath.Join(config.SystemConfigDir(), pidginConfigDir)

	if fi, err := os.Stat(filepath.Join(app, pidginAccountsFile)); err == nil && !fi.IsDir() {
		return app, true
	}

	return "", false
}
