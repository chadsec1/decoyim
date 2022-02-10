// +build !windows

package config

import "os/user"

// IsWindows returns true if this is running under windows
func IsWindows() bool {
	return false
}

// SystemConfigDir points to the function that gets the configuration directory for this system
var SystemConfigDir = XdgConfigHome

// SystemDataDir points to the function that gets the data directory for this system
var SystemDataDir = XdgDataHome

// // SystemConfigDir returns the application data directory, valid on both windows and posix systems
// func SystemConfigDir() string {
// 	//TODO: Why not use g_get_user_config_dir()?
// 	// https://developer.gnome.org/glib/unstable/glib-Miscellaneous-Utility-Functions.html#g-get-user-config-dir
// 	return XdgConfigHome()
// }

func localHome() string {
	u, e := user.Current()
	if e == nil {
		return u.HomeDir
	}
	return ""
}
