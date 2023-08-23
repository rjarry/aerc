package xdg

import (
	"errors"
	"os/user"
	"testing"
)

func TestHomeDir(t *testing.T) {
	t.Run("from env", func(t *testing.T) {
		t.Setenv("HOME", "/home/user")
		home := HomeDir()
		if home != "/home/user" {
			t.Errorf(`got %q expected "/home/user"`, home)
		}
	})
	t.Run("from getpwuid_r", func(t *testing.T) {
		t.Setenv("HOME", "")
		orig := currentUser
		currentUser = func() (*user.User, error) {
			return &user.User{HomeDir: "/home/user"}, nil
		}
		home := HomeDir()
		currentUser = orig
		if home != "/home/user" {
			t.Errorf(`got %q expected "/home/user"`, home)
		}
	})
	t.Run("failure", func(t *testing.T) {
		t.Setenv("HOME", "")
		orig := currentUser
		currentUser = func() (*user.User, error) {
			return nil, errors.New("no such user")
		}
		home := HomeDir()
		currentUser = orig
		if home != "" {
			t.Errorf(`got %q expected ""`, home)
		}
	})
}

func TestExpandHome(t *testing.T) {
	t.Setenv("HOME", "/home/user")
	vectors := []struct {
		args     []string
		expected string
	}{
		{args: []string{"foo"}, expected: "foo"},
		{args: []string{"foo", "bar"}, expected: "foo/bar"},
		{args: []string{"/foobar/baz"}, expected: "/foobar/baz"},
		{args: []string{"~/foobar/baz"}, expected: "/home/user/foobar/baz"},
		{args: []string{}, expected: ""},
		{args: []string{"~"}, expected: "/home/user"},
	}
	for _, vec := range vectors {
		t.Run(vec.expected, func(t *testing.T) {
			res := ExpandHome(vec.args...)
			if res != vec.expected {
				t.Errorf("got %q expected %q", res, vec.expected)
			}
		})
	}
}

func TestTildeHome(t *testing.T) {
	t.Setenv("HOME", "/home/user")
	vectors := []struct {
		arg      string
		expected string
	}{
		{arg: "foo", expected: "foo"},
		{arg: "foo/bar", expected: "foo/bar"},
		{arg: "/foobar/baz", expected: "/foobar/baz"},
		{arg: "/home/user/foobar/baz", expected: "~/foobar/baz"},
		{arg: "", expected: ""},
		{arg: "/home/user", expected: "~"},
		{arg: "/home/user2/foobar/baz", expected: "/home/user2/foobar/baz"},
	}
	for _, vec := range vectors {
		t.Run(vec.expected, func(t *testing.T) {
			res := TildeHome(vec.arg)
			if res != vec.expected {
				t.Errorf("got %q expected %q", res, vec.expected)
			}
		})
	}
}
