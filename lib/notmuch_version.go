//go:build notmuch

package lib

import "git.sr.ht/~rjarry/aerc/lib/notmuch"

func NotmuchVersion() (string, bool) {
	return notmuch.Version(), true
}
