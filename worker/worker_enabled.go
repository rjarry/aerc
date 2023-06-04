package worker

// the following workers are always enabled
import (
	_ "git.sr.ht/~rjarry/aerc/worker/imap"
	_ "git.sr.ht/~rjarry/aerc/worker/jmap"
	_ "git.sr.ht/~rjarry/aerc/worker/lib/watchers"
	_ "git.sr.ht/~rjarry/aerc/worker/maildir"
	_ "git.sr.ht/~rjarry/aerc/worker/mbox"
)
