package completer

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/mail"
	"os/exec"
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
	logger     *log.Logger
}

// A CompleteFunc accepts a string to be completed and returns a slice of
// possible completions.
type CompleteFunc func(string) []string

// New creates a new Completer with the specified address book command.
func New(addressBookCmd string, errHandler func(error), logger *log.Logger) *Completer {
	return &Completer{
		AddressBookCmd: addressBookCmd,
		errHandler:     errHandler,
		logger:         logger,
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
		return func(s string) []string {
			completions, err := c.completeAddress(s)
			if err != nil {
				c.handleErr(err)
				return []string{}
			}
			return completions
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
// completions for the specified string, returning a slice of completions or an
// error.
func (c *Completer) completeAddress(s string) ([]string, error) {
	cmd, err := c.getAddressCmd(s)
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout: %v", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("cmd start: %v", err)
	}
	completions, err := readCompletions(stdout)
	if err != nil {
		return nil, fmt.Errorf("read completions: %v", err)
	}

	// Wait returns an error if the exit status != 0, which some completion
	// programs will do to signal no matches. We don't want to spam the user with
	// spurious error messages, so we'll ignore any errors that arise at this
	// point.
	if err := cmd.Wait(); err != nil {
		c.logger.Printf("completion error: %v", err)
	}

	return completions, nil
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
		if addr, err := mail.ParseAddress(parts[0]); err == nil {
			if len(parts) > 1 {
				addr.Name = parts[1]
			}
			completions = append(completions, addr.String())
		}
	}
}

func (c *Completer) handleErr(err error) {
	if c.errHandler != nil {
		c.errHandler(err)
	}
}
