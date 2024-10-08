package xdg

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

// Return a path relative to the user home cache dir
func CachePath(paths ...string) string {
	res := filepath.Join(paths...)
	if !filepath.IsAbs(res) {
		var cache string
		if runtime.GOOS == "darwin" {
			// preserve backward compat with github.com/kyoh86/xdg
			cache = os.Getenv("XDG_CACHE_HOME")
		}
		if cache == "" {
			var err error
			cache, err = os.UserCacheDir()
			if err != nil {
				cache = ExpandHome("~/.cache")
			}
		}
		res = filepath.Join(cache, res)
	}
	return res
}

// Return a path relative to the user home config dir
func ConfigPath(paths ...string) string {
	res := filepath.Join(paths...)
	if !filepath.IsAbs(res) {
		var config string
		if runtime.GOOS == "darwin" {
			// preserve backward compat with github.com/kyoh86/xdg
			config = os.Getenv("XDG_CONFIG_HOME")
			if config == "" {
				config = ExpandHome("~/Library/Preferences")
			}
		} else {
			var err error
			config, err = os.UserConfigDir()
			if err != nil {
				config = ExpandHome("~/.config")
			}
		}
		res = filepath.Join(config, res)
	}
	return res
}

// Return a path relative to the user data home dir
func DataPath(paths ...string) string {
	res := filepath.Join(paths...)
	if !filepath.IsAbs(res) {
		data := os.Getenv("XDG_DATA_HOME")
		// preserve backward compat with github.com/kyoh86/xdg
		if data == "" && runtime.GOOS == "darwin" {
			data = ExpandHome("~/Library/Application Support")
		} else if data == "" {
			data = ExpandHome("~/.local/share")
		}
		res = filepath.Join(data, res)
	}
	return res
}

// ugly: there's no other way to allow mocking a function in go...
var userRuntimePath = func() string {
	uid := strconv.Itoa(os.Getuid())
	path := filepath.Join("/run/user", uid)
	fi, err := os.Stat(path)
	if err != nil || !fi.Mode().IsDir() {
		// OpenRC does not create /run/user. TMUX and Neovim
		// create /tmp/$program-$uid instead. Mimic that.
		path = filepath.Join(os.TempDir(), "aerc-"+uid)
		err = os.MkdirAll(path, 0o700)
		if err != nil {
			// Fallback to /tmp if all else fails.
			path = os.TempDir()
		}
	}
	return path
}

// Return a path relative to the user runtime dir
func RuntimePath(paths ...string) string {
	res := filepath.Join(paths...)
	if !filepath.IsAbs(res) {
		run := os.Getenv("XDG_RUNTIME_DIR")
		// preserve backward compat with github.com/kyoh86/xdg
		if run == "" && runtime.GOOS == "darwin" {
			run = ExpandHome("~/Library/Application Support")
		} else if run == "" {
			run = userRuntimePath()
		}
		res = filepath.Join(run, res)
	}
	return res
}
