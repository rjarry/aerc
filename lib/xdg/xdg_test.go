package xdg

import (
	"runtime"
	"testing"
)

func TestCachePath(t *testing.T) {
	t.Setenv("HOME", "/home/user")
	vectors := []struct {
		args     []string
		env      map[string]string
		expected map[string]string
	}{
		{
			args: []string{"aerc", "foo", "history"},
			expected: map[string]string{
				"":       "/home/user/.cache/aerc/foo/history",
				"darwin": "/home/user/Library/Caches/aerc/foo/history",
			},
		},
		{
			args:     []string{"aerc", "foo/zuul"},
			env:      map[string]string{"XDG_CACHE_HOME": "/home/x/.cache"},
			expected: map[string]string{"": "/home/x/.cache/aerc/foo/zuul"},
		},
		{
			args:     []string{},
			env:      map[string]string{"XDG_CACHE_HOME": "/blah"},
			expected: map[string]string{"": "/blah"},
		},
	}
	for _, vec := range vectors {
		expected, found := vec.expected[runtime.GOOS]
		if !found {
			expected = vec.expected[""]
		}
		t.Run(expected, func(t *testing.T) {
			for key, value := range vec.env {
				t.Setenv(key, value)
			}
			res := CachePath(vec.args...)
			if res != expected {
				t.Errorf("got %q expected %q", res, expected)
			}
		})
	}
}

func TestConfigPath(t *testing.T) {
	t.Setenv("HOME", "/home/user")
	vectors := []struct {
		args     []string
		env      map[string]string
		expected map[string]string
	}{
		{
			args: []string{"aerc", "accounts.conf"},
			expected: map[string]string{
				"":       "/home/user/.config/aerc/accounts.conf",
				"darwin": "/home/user/Library/Preferences/aerc/accounts.conf",
			},
		},
		{
			args:     []string{"aerc", "accounts.conf"},
			env:      map[string]string{"XDG_CONFIG_HOME": "/users/x/.config"},
			expected: map[string]string{"": "/users/x/.config/aerc/accounts.conf"},
		},
		{
			args:     []string{},
			env:      map[string]string{"XDG_CONFIG_HOME": "/blah"},
			expected: map[string]string{"": "/blah"},
		},
	}
	for _, vec := range vectors {
		expected, found := vec.expected[runtime.GOOS]
		if !found {
			expected = vec.expected[""]
		}
		t.Run(expected, func(t *testing.T) {
			for key, value := range vec.env {
				t.Setenv(key, value)
			}
			res := ConfigPath(vec.args...)
			if res != expected {
				t.Errorf("got %q expected %q", res, expected)
			}
		})
	}
}

func TestDataPath(t *testing.T) {
	t.Setenv("HOME", "/home/user")
	vectors := []struct {
		args     []string
		env      map[string]string
		expected map[string]string
	}{
		{
			args: []string{"aerc", "templates"},
			expected: map[string]string{
				"":       "/home/user/.local/share/aerc/templates",
				"darwin": "/home/user/Library/Application Support/aerc/templates",
			},
		},
		{
			args:     []string{"aerc", "templates"},
			env:      map[string]string{"XDG_DATA_HOME": "/users/x/.local/share"},
			expected: map[string]string{"": "/users/x/.local/share/aerc/templates"},
		},
		{
			args:     []string{},
			env:      map[string]string{"XDG_DATA_HOME": "/blah"},
			expected: map[string]string{"": "/blah"},
		},
	}
	for _, vec := range vectors {
		expected, found := vec.expected[runtime.GOOS]
		if !found {
			expected = vec.expected[""]
		}
		t.Run(expected, func(t *testing.T) {
			for key, value := range vec.env {
				t.Setenv(key, value)
			}
			res := DataPath(vec.args...)
			if res != expected {
				t.Errorf("got %q expected %q", res, expected)
			}
		})
	}
}

func TestRuntimePath(t *testing.T) {
	// poor man's function mocking
	orig := userRuntimePath
	userRuntimePath = func() string { return "/run/user/1000" }
	defer func() { userRuntimePath = orig }()
	t.Setenv("HOME", "/home/user")

	vectors := []struct {
		args     []string
		env      map[string]string
		expected map[string]string
	}{
		{
			args: []string{"aerc.sock"},
			expected: map[string]string{
				"":       "/run/user/1000/aerc.sock",
				"darwin": "/home/user/Library/Application Support/aerc.sock",
			},
		},
		{
			args:     []string{"aerc.sock"},
			env:      map[string]string{"XDG_RUNTIME_DIR": "/run/user/1234"},
			expected: map[string]string{"": "/run/user/1234/aerc.sock"},
		},
		{
			args:     []string{},
			env:      map[string]string{"XDG_RUNTIME_DIR": "/blah"},
			expected: map[string]string{"": "/blah"},
		},
	}
	for _, vec := range vectors {
		expected, found := vec.expected[runtime.GOOS]
		if !found {
			expected = vec.expected[""]
		}
		t.Run(expected, func(t *testing.T) {
			for key, value := range vec.env {
				t.Setenv(key, value)
			}
			res := RuntimePath(vec.args...)
			if res != expected {
				t.Errorf("got %q expected %q", res, expected)
			}
		})
	}
}
