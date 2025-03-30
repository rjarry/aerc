package extensions

import (
	"fmt"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-imap/responses"
	"github.com/emersion/go-imap/utf7"
)

// A LIST-STATUS client
type ListStatusClient struct {
	c *client.Client
}

func NewListStatusClient(c *client.Client) *ListStatusClient {
	return &ListStatusClient{c}
}

// SupportListStatus checks if the server supports the LIST-STATUS extension.
func (c *ListStatusClient) SupportListStatus() (bool, error) {
	return c.c.Support("LIST-STATUS")
}

// ListStatus performs a LIST-STATUS command, listing mailboxes and also
// retrieving the requested status items. A nil channel can be passed in order
// to only retrieve the STATUS responses
func (c *ListStatusClient) ListStatus(
	ref string,
	name string,
	items []imap.StatusItem,
	ch chan *imap.MailboxInfo,
) ([]*imap.MailboxStatus, error) {
	if ch != nil {
		defer close(ch)
	}

	if c.c.State() != imap.AuthenticatedState && c.c.State() != imap.SelectedState {
		return nil, client.ErrNotLoggedIn
	}

	cmd := &ListStatusCommand{
		Reference: ref,
		Mailbox:   name,
		Items:     items,
	}
	res := &ListStatusResponse{Mailboxes: ch}

	status, err := c.c.Execute(cmd, res)
	if err != nil {
		return nil, err
	}
	return res.Statuses, status.Err()
}

// ListStatusCommand is a LIST command, as defined in RFC 3501 section 6.3.8. If
// Subscribed is set to true, LSUB will be used instead. Mailbox statuses will
// be returned if Items is not nil
type ListStatusCommand struct {
	Reference string
	Mailbox   string

	Subscribed bool
	Items      []imap.StatusItem
}

func (cmd *ListStatusCommand) Command() *imap.Command {
	name := "LIST"
	if cmd.Subscribed {
		name = "LSUB"
	}

	enc := utf7.Encoding.NewEncoder()
	ref, _ := enc.String(cmd.Reference)
	mailbox, _ := enc.String(cmd.Mailbox)

	items := make([]string, len(cmd.Items))
	if cmd.Items != nil {
		for i, item := range cmd.Items {
			items[i] = string(item)
		}
	}

	args := fmt.Sprintf("RETURN (STATUS (%s))", strings.Join(items, " "))
	return &imap.Command{
		Name:      name,
		Arguments: []any{ref, mailbox, imap.RawString(args)},
	}
}

// A LIST-STATUS response
type ListStatusResponse struct {
	Mailboxes  chan *imap.MailboxInfo
	Subscribed bool
	Statuses   []*imap.MailboxStatus
}

func (r *ListStatusResponse) Name() string {
	if r.Subscribed {
		return "LSUB"
	} else {
		return "LIST"
	}
}

func (r *ListStatusResponse) Handle(resp imap.Resp) error {
	name, _, ok := imap.ParseNamedResp(resp)
	if !ok {
		return responses.ErrUnhandled
	}
	switch name {
	case "LIST":
		if r.Mailboxes == nil {
			return nil
		}
		res := responses.List{Mailboxes: r.Mailboxes}
		return res.Handle(resp)
	case "STATUS":
		res := responses.Status{
			Mailbox: new(imap.MailboxStatus),
		}
		err := res.Handle(resp)
		if err != nil {
			return err
		}
		r.Statuses = append(r.Statuses, res.Mailbox)
	default:
		return responses.ErrUnhandled
	}

	return nil
}

func (r *ListStatusResponse) WriteTo(w *imap.Writer) error {
	respName := r.Name()

	for mbox := range r.Mailboxes {
		fields := []any{imap.RawString(respName)}
		fields = append(fields, mbox.Format()...)

		resp := imap.NewUntaggedResp(fields)
		if err := resp.WriteTo(w); err != nil {
			return err
		}
	}
	return nil
}
