package completer

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/mail"
	"os/exec"
	"regexp"
	"strings"

	"github.com/google/shlex"
)

// A Completer is used to autocomplete text inputs based on the configured
// completion commands.
type Completer struct {
	// AddressBookCmd is the command to run for completing email addresses. This
	// command must output one completion on each line with fields separated by a
	// tab character. The first field must be the address, and the second field,
	// if present, the contact name. Only the email address field is required.
	// The name field is optional. Additional fields are ignored.
	AddressBookCmd string

	errHandler func(error)
}

// A CompleteFunc accepts a string to be completed and returns a slice of
// completions candidates with a prefix to prepend to the chosen candidate
type CompleteFunc func(string) ([]string, string)

// New creates a new Completer with the specified address book command.
func New(addressBookCmd string, errHandler func(error)) *Completer {
	return &Completer{
		AddressBookCmd: addressBookCmd,
		errHandler:     errHandler,
	}
}

// ForHeader returns a CompleteFunc appropriate for the specified mail header. In
// the case of To, From, etc., the completer will get completions from the
// configured address book command. For other headers, a noop completer will be
// returned. If errors arise during completion, the errHandler will be called.
func (c *Completer) ForHeader(h string) CompleteFunc {
	if isAddressHeader(h) {
		if c.AddressBookCmd == "" {
			return nil
		}
		// wrap completeAddress in an error handler
		return func(s string) ([]string, string) {
			completions, prefix, err := c.completeAddress(s)
			if err != nil {
				c.handleErr(err)
				return []string{}, ""
			}
			return completions, prefix
		}
	}
	return nil
}

// isAddressHeader determines whether the address completer should be used for
// header h.
func isAddressHeader(h string) bool {
	switch strings.ToLower(h) {
	case "to", "from", "cc", "bcc":
		return true
	}
	return false
}

// completeAddress uses the configured address book completion command to fetch
// completions for the specified string, returning a slice of completions and
// a prefix to be prepended to the selected completion, or an error.
func (c *Completer) completeAddress(s string) ([]string, string, error) {
	prefix, candidate := c.parseAddress(s)
	cmd, err := c.getAddressCmd(candidate)
	if err != nil {
		return nil, "", err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, "", fmt.Errorf("stdout: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, "", fmt.Errorf("stderr: %v", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, "", fmt.Errorf("cmd start: %v", err)
	}
	// Wait returns an error if the exit status != 0, which some completion
	// programs will do to signal no matches. We don't want to spam the user with
	// spurious error messages, so we'll ignore any errors that arise at this
	// point.
	defer cmd.Wait() //nolint:errcheck // see above

	completions, err := readCompletions(stdout)
	if err != nil {
		buf, _ := ioutil.ReadAll(stderr)
		msg := strings.TrimSpace(string(buf))
		if msg != "" {
			msg = ": " + msg
		}
		return nil, "", fmt.Errorf("read completions%s: %v", msg, err)
	}

	return completions, prefix, nil
}

// parseAddress will break an address header into a prefix (containing
// the already valid addresses) and an input for completion
func (c *Completer) parseAddress(s string) (string, string) {
	pattern := regexp.MustCompile(`^(.*),\s+([^,]*)$`)
	matches := pattern.FindStringSubmatch(s)
	if matches == nil {
		return "", s
	}
	return matches[1] + ", ", matches[2]
}

// getAddressCmd constructs an exec.Cmd based on the configured command and
// specified query.
func (c *Completer) getAddressCmd(s string) (*exec.Cmd, error) {
	if strings.TrimSpace(c.AddressBookCmd) == "" {
		return nil, fmt.Errorf("no command configured")
	}
	queryCmd := strings.Replace(c.AddressBookCmd, "%s", s, -1)
	parts, err := shlex.Split(queryCmd)
	if err != nil {
		return nil, fmt.Errorf("could not lex command")
	}
	if len(parts) < 1 {
		return nil, fmt.Errorf("empty command")
	}
	if len(parts) > 1 {
		return exec.Command(parts[0], parts[1:]...), nil
	}
	return exec.Command(parts[0]), nil
}

// readCompletions reads a slice of completions from r line by line. Each line
// must consist of tab-delimited fields. Only the first field (the email
// address field) is required, the second field (the contact name) is optional,
// and subsequent fields are ignored.
func readCompletions(r io.Reader) ([]string, error) {
	buf := bufio.NewReader(r)
	completions := []string{}
	for {
		line, err := buf.ReadString('\n')
		if err == io.EOF {
			return completions, nil
		} else if err != nil {
			return nil, err
		}
		parts := strings.SplitN(line, "\t", 3)
		addr, err := mail.ParseAddress(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, err
		}
		if len(parts) > 1 {
			addr.Name = strings.TrimSpace(parts[1])
		}
		decoded, err := decodeMIME(addr.String())
		if err != nil {
			return nil, fmt.Errorf("could not decode MIME string: %w", err)
		}
		completions = append(completions, decoded)
	}
}

func decodeMIME(s string) (string, error) {
	var d mime.WordDecoder
	return d.DecodeHeader(s)
}

func (c *Completer) handleErr(err error) {
	if c.errHandler != nil {
		c.errHandler(err)
	}
}
