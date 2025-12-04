package completer

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"os/exec"
	"regexp"
	"strings"
	"syscall"

	"git.sr.ht/~rjarry/aerc/lib/format"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/go-opt/v2"
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
type CompleteFunc func(context.Context, string) ([]opt.Completion, string)

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
		return func(ctx context.Context, s string) ([]opt.Completion, string) {
			completions, prefix, err := c.completeAddress(ctx, s)
			if err != nil {
				c.handleErr(err)
				return []opt.Completion{}, ""
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
	case "to", "from", "cc", "bcc", "reply-to":
		return true
	}
	return false
}

const maxCompletionLines = 100

var tooManyLines = fmt.Errorf("returned more than %d lines", maxCompletionLines)

// completeAddress uses the configured address book completion command to fetch
// completions for the specified string, returning a slice of completions and
// a prefix to be prepended to the selected completion, or an error.
func (c *Completer) completeAddress(ctx context.Context, s string) ([]opt.Completion, string, error) {
	prefix, candidate := c.parseAddress(s)
	cmd, err := c.getAddressCmd(ctx, candidate)
	if err != nil {
		return nil, "", err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, "", fmt.Errorf("stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, "", fmt.Errorf("stderr: %w", err)
	}
	// reset the process group id to allow killing all its children
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return nil, "", fmt.Errorf("cmd start: %w", err)
	}
	// Wait returns an error if the exit status != 0, which some completion
	// programs will do to signal no matches. We don't want to spam the user with
	// spurious error messages, so we'll ignore any errors that arise at this
	// point.
	defer cmd.Wait() //nolint:errcheck // see above

	completions, err := readCompletions(stdout)
	if err != nil {
		// make sure to kill the process *and* all its children
		//nolint:errcheck // who cares?
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		log.Warnf("command %s killed: %s", cmd, err)
	}
	if err != nil && !errors.Is(err, tooManyLines) {
		buf, _ := io.ReadAll(stderr)
		msg := strings.TrimSpace(string(buf))
		if msg != "" {
			msg = ": " + msg
		}
		return nil, "", fmt.Errorf("read completions%s: %w", msg, err)
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
func (c *Completer) getAddressCmd(ctx context.Context, s string) (*exec.Cmd, error) {
	if strings.TrimSpace(c.AddressBookCmd) == "" {
		return nil, fmt.Errorf("no command configured")
	}
	queryCmd := strings.ReplaceAll(c.AddressBookCmd, "%s", s)
	return exec.CommandContext(ctx, "sh", "-c", queryCmd), nil
}

// readCompletions reads a slice of completions from r line by line. Each line
// must consist of tab-delimited fields. Only the first field (the email
// address field) is required, the second field (the contact name) is optional,
// and subsequent fields are ignored.
func readCompletions(r io.Reader) ([]opt.Completion, error) {
	buf := bufio.NewReader(r)
	var completions []opt.Completion
	for i := 0; i < maxCompletionLines; i++ {
		line, err := buf.ReadString('\n')
		if errors.Is(err, io.EOF) {
			return completions, nil
		} else if err != nil {
			return nil, err
		}
		if strings.TrimSpace(line) == "" {
			// skip empty lines
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		addr, err := mail.ParseAddress(strings.TrimSpace(parts[0]))
		if err != nil {
			log.Warnf("line %d: %#v: could not parse address: %v",
				line, err)
			continue
		}
		if len(parts) > 1 {
			addr.Name = strings.TrimSpace(parts[1])
		}
		completions = append(completions, opt.Completion{
			Value: format.AddressForHumans(addr),
		})
	}
	return completions, tooManyLines
}

func (c *Completer) handleErr(err error) {
	if c.errHandler != nil {
		c.errHandler(err)
	}
}
