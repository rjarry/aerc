package extensions

import (
	"errors"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-imap/responses"
)

type IDClient struct {
	c *client.Client
}

func NewIDClient(c *client.Client) *IDClient {
	return &IDClient{c}
}

// SupportID checks if the server supports the ID extension (RFC 2971).
func (c *IDClient) SupportID() (bool, error) {
	return c.c.Support("ID")
}

// ID sends an IMAP ID command (RFC 2971). Required by NetEase (163/126/yeah.net).
// Common keys: name, version, vendor, support-email.
// If params is empty, NIL is sent to query the server's ID.
func (c *IDClient) ID(params map[string]string) (map[string]string, error) {
	if c.c.State()&imap.ConnectedState == 0 {
		return nil, client.ErrNotLoggedIn
	}

	cmd := &IDCommand{Params: params}
	res := &IDResponse{}

	status, err := c.c.Execute(cmd, res)
	if err != nil {
		return nil, err
	}

	return res.Params, status.Err()
}

type IDCommand struct {
	Params map[string]string
}

func (cmd *IDCommand) Command() *imap.Command {
	var args []any
	if len(cmd.Params) == 0 {
		args = []any{nil}
	} else {
		var pairs []any
		for k, v := range cmd.Params {
			pairs = append(pairs, k)
			pairs = append(pairs, v)
		}
		args = []any{pairs}
	}
	return &imap.Command{
		Name:      "ID",
		Arguments: args,
	}
}

type IDResponse struct {
	Params map[string]string
}

func (r *IDResponse) Handle(resp imap.Resp) error {
	name, fields, ok := imap.ParseNamedResp(resp)
	if !ok || name != "ID" {
		return responses.ErrUnhandled
	}

	if len(fields) == 0 {
		return nil
	}

	if len(fields) == 1 {
		if s, ok := fields[0].(string); ok && strings.EqualFold(s, "NIL") {
			return nil
		}
	}

	var pairs []any
	for _, f := range fields {
		pairs = append(pairs, f)
	}

	if len(pairs)%2 != 0 {
		return errors.New("imap: odd number of fields in ID response")
	}

	r.Params = make(map[string]string, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		key, ok1 := pairs[i].(string)
		val, ok2 := pairs[i+1].(string)
		if ok1 && ok2 {
			r.Params[key] = val
		}
	}

	return nil
}
