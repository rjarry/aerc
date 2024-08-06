package pinentry

import (
	"fmt"
	"os"
	"strings"

	"git.sr.ht/~rjarry/aerc/lib/log"
)

var missingGPGTTYmsg = `
You need to set GPG_TTY manually before starting aerc. Add the following to your
.bashrc or whatever initialization file is used for shell invocations:

	GPG_TTY=$(tty)
	export GPG_TTY

Further information can be found here:
https://www.gnupg.org/documentation/manuals/gnupg/Invoking-GPG_002dAGENT.html
`

// ttyname returns current name of the pty. This is necessary in order to tell
// pinentry where to ask for the passphrase.
//
// If there is a GPG_TTY environment variable set, use this one. Otherwise, try
// readline() on /proc/<pid>/fd/0.
//
// If both approaches fail, the user's only option is to set GPG_TTY manually.
//
// If tty name could not be determined, an empty string is returned.
func ttyname() string {
	if s := os.Getenv("GPG_TTY"); s != "" {
		return s
	}

	// try readlink or else show missing GPG_TTY warning msg
	tty, err := os.Readlink(fmt.Sprintf("/proc/%d/fd/0", os.Getpid()))
	if err != nil {
		log.Debugf("readlink: '%s' with err: %v", tty, err)
		log.Warnf(missingGPGTTYmsg)
		return ""
	}

	return strings.TrimSpace(tty)
}
