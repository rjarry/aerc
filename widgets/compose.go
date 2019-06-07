package widgets

import (
	"io"
	"io/ioutil"
	gomail "net/mail"
	"os"
	"os/exec"
	"time"

	"github.com/emersion/go-message"
	"github.com/emersion/go-message/mail"
	"github.com/gdamore/tcell"
	"github.com/mattn/go-runewidth"
	"github.com/pkg/errors"

	"git.sr.ht/~sircmpwn/aerc/config"
	"git.sr.ht/~sircmpwn/aerc/lib"
	"git.sr.ht/~sircmpwn/aerc/lib/ui"
	"git.sr.ht/~sircmpwn/aerc/worker/types"
)

type Composer struct {
	headers struct {
		from    *headerEditor
		subject *headerEditor
		to      *headerEditor
	}

	acct   *config.AccountConfig
	config *config.AercConfig

	defaults map[string]string
	editor   *Terminal
	email    *os.File
	grid     *ui.Grid
	review   *reviewMessage
	worker   *types.Worker

	focusable []ui.DrawableInteractive
	focused   int
}

// TODO: Let caller configure headers, initial body (for replies), etc
func NewComposer(conf *config.AercConfig,
	acct *config.AccountConfig, worker *types.Worker) *Composer {

	grid := ui.NewGrid().Rows([]ui.GridSpec{
		{ui.SIZE_EXACT, 3},
		{ui.SIZE_WEIGHT, 1},
	}).Columns([]ui.GridSpec{
		{ui.SIZE_WEIGHT, 1},
	})

	// TODO: let user specify extra headers to edit by default
	headers := ui.NewGrid().Rows([]ui.GridSpec{
		{ui.SIZE_EXACT, 1}, // To/From
		{ui.SIZE_EXACT, 1}, // Subject
		{ui.SIZE_EXACT, 1}, // [spacer]
	}).Columns([]ui.GridSpec{
		{ui.SIZE_WEIGHT, 1},
		{ui.SIZE_WEIGHT, 1},
	})

	to := newHeaderEditor("To", "")
	from := newHeaderEditor("From", acct.From)
	subject := newHeaderEditor("Subject", "")
	headers.AddChild(to).At(0, 0)
	headers.AddChild(from).At(0, 1)
	headers.AddChild(subject).At(1, 0).Span(1, 2)
	headers.AddChild(ui.NewFill(' ')).At(2, 0).Span(1, 2)

	email, err := ioutil.TempFile("", "aerc-compose-*.eml")
	if err != nil {
		// TODO: handle this better
		return nil
	}

	grid.AddChild(headers).At(0, 0)

	c := &Composer{
		acct:   acct,
		config: conf,
		email:  email,
		grid:   grid,
		worker: worker,
		// You have to backtab to get to "From", since you usually don't edit it
		focused:   1,
		focusable: []ui.DrawableInteractive{from, to, subject},
	}
	c.headers.to = to
	c.headers.from = from
	c.headers.subject = subject
	c.ShowTerminal()

	return c
}

// Sets additional headers to be added to the outgoing email (e.g. In-Reply-To)
func (c *Composer) Defaults(defaults map[string]string) *Composer {
	c.defaults = defaults
	if to, ok := defaults["To"]; ok {
		c.headers.to.input.Set(to)
		delete(defaults, "To")
	}
	if from, ok := defaults["From"]; ok {
		c.headers.from.input.Set(from)
		delete(defaults, "From")
	}
	if subject, ok := defaults["Subject"]; ok {
		c.headers.subject.input.Set(subject)
		delete(defaults, "Subject")
	}
	return c
}

// Note: this does not reload the editor. You must call this before the first
// Draw() call.
func (c *Composer) SetContents(reader io.Reader) *Composer {
	c.email.Seek(0, os.SEEK_SET)
	io.Copy(c.email, reader)
	c.email.Sync()
	c.email.Seek(0, os.SEEK_SET)
	return c
}

func (c *Composer) FocusTerminal() *Composer {
	if c.editor == nil {
		return c
	}
	c.focusable[c.focused].Focus(false)
	c.focused = 3
	c.focusable[c.focused].Focus(true)
	return c
}

func (c *Composer) OnSubjectChange(fn func(subject string)) {
	c.headers.subject.OnChange(func() {
		fn(c.headers.subject.input.String())
	})
}

func (c *Composer) Draw(ctx *ui.Context) {
	c.grid.Draw(ctx)
}

func (c *Composer) Invalidate() {
	c.grid.Invalidate()
}

func (c *Composer) OnInvalidate(fn func(d ui.Drawable)) {
	c.grid.OnInvalidate(func(_ ui.Drawable) {
		fn(c)
	})
}

func (c *Composer) Close() {
	if c.email != nil {
		path := c.email.Name()
		c.email.Close()
		os.Remove(path)
		c.email = nil
	}
	if c.editor != nil {
		c.editor.Destroy()
		c.editor = nil
	}
}

func (c *Composer) Bindings() string {
	if c.editor == nil {
		return "compose::review"
	} else if c.editor == c.focusable[c.focused] {
		return "compose::editor"
	} else {
		return "compose"
	}
}

func (c *Composer) Event(event tcell.Event) bool {
	return c.focusable[c.focused].Event(event)
}

func (c *Composer) Focus(focus bool) {
	c.focusable[c.focused].Focus(focus)
}

func (c *Composer) Config() *config.AccountConfig {
	return c.acct
}

func (c *Composer) Worker() *types.Worker {
	return c.worker
}

func (c *Composer) PrepareHeader() (*mail.Header, []string, error) {
	// Extract headers from the email, if present
	c.email.Seek(0, os.SEEK_SET)
	var (
		rcpts  []string
		header mail.Header
	)
	reader, err := mail.CreateReader(c.email)
	if err == nil {
		header = reader.Header
		defer reader.Close()
	} else {
		c.email.Seek(0, os.SEEK_SET)
	}
	// Update headers
	mhdr := (*message.Header)(&header.Header)
	mhdr.SetContentType("text/plain", map[string]string{"charset": "UTF-8"})
	mhdr.SetText("Message-Id", lib.GenerateMessageId())
	if subject, _ := header.Subject(); subject == "" {
		header.SetSubject(c.headers.subject.input.String())
	}
	if date, err := header.Date(); err != nil || date == (time.Time{}) {
		header.SetDate(time.Now())
	}
	from := c.headers.from.input.String()
	from_addrs, err := gomail.ParseAddressList(from)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "ParseAddressList(%s)", from)
	} else {
		var simon_from []*mail.Address
		for _, addr := range from_addrs {
			simon_from = append(simon_from, (*mail.Address)(addr))
		}
		header.SetAddressList("From", simon_from)
	}
	if to := c.headers.to.input.String(); to != "" {
		// Dammit Simon, this branch is 3x as long as it ought to be because
		// your types aren't compatible enough with each other
		to_rcpts, err := gomail.ParseAddressList(to)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "ParseAddressList(%s)", to)
		}
		ed_rcpts, err := header.AddressList("To")
		if err != nil {
			return nil, nil, errors.Wrap(err, "AddressList(To)")
		}
		for _, addr := range to_rcpts {
			ed_rcpts = append(ed_rcpts, (*mail.Address)(addr))
		}
		header.SetAddressList("To", ed_rcpts)
		for _, addr := range ed_rcpts {
			rcpts = append(rcpts, addr.Address)
		}
	}
	if cc, _ := mhdr.Text("Cc"); cc != "" {
		cc_rcpts, err := gomail.ParseAddressList(cc)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "ParseAddressList(%s)", cc)
		}
		// TODO: Update when the user inputs Cc's through the UI
		for _, addr := range cc_rcpts {
			rcpts = append(rcpts, addr.Address)
		}
	}
	if bcc, _ := mhdr.Text("Bcc"); bcc != "" {
		bcc_rcpts, err := gomail.ParseAddressList(bcc)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "ParseAddressList(%s)", bcc)
		}
		// TODO: Update when the user inputs Bcc's through the UI
		for _, addr := range bcc_rcpts {
			rcpts = append(rcpts, addr.Address)
		}
	}
	// Merge in additional headers
	txthdr := mhdr.Header
	for key, value := range c.defaults {
		if !txthdr.Has(key) && value != "" {
			mhdr.SetText(key, value)
		}
	}
	return &header, rcpts, nil
}

func (c *Composer) WriteMessage(header *mail.Header, writer io.Writer) error {
	c.email.Seek(0, os.SEEK_SET)
	var body io.Reader
	reader, err := mail.CreateReader(c.email)
	if err == nil {
		// TODO: Do we want to let users write a full blown multipart email
		// into the editor? If so this needs to change
		part, err := reader.NextPart()
		if err != nil {
			return errors.Wrap(err, "reader.NextPart")
		}
		body = part.Body
		defer reader.Close()
	} else {
		c.email.Seek(0, os.SEEK_SET)
		body = c.email
	}
	// TODO: attachments
	w, err := mail.CreateSingleInlineWriter(writer, *header)
	if err != nil {
		return errors.Wrap(err, "CreateSingleInlineWriter")
	}
	defer w.Close()
	if _, err := io.Copy(w, body); err != nil {
		return errors.Wrap(err, "io.Copy")
	}
	return nil
}

func (c *Composer) termClosed(err error) {
	c.grid.RemoveChild(c.editor)
	c.review = newReviewMessage(c, err)
	c.grid.AddChild(c.review).At(1, 0)
	c.editor.Destroy()
	c.editor = nil
	c.focusable = c.focusable[:len(c.focusable)-1]
	if c.focused >= len(c.focusable) {
		c.focused = len(c.focusable) - 1
	}
}

func (c *Composer) ShowTerminal() {
	if c.editor != nil {
		return
	}
	if c.review != nil {
		c.grid.RemoveChild(c.review)
	}
	editorName := c.config.Compose.Editor
	if editorName == "" {
		editorName = os.Getenv("EDITOR")
	}
	if editorName == "" {
		editorName = "vi"
	}
	editor := exec.Command("/bin/sh", "-c", editorName+" "+c.email.Name())
	c.editor, _ = NewTerminal(editor) // TODO: handle error
	c.editor.OnClose = c.termClosed
	c.grid.AddChild(c.editor).At(1, 0)
	c.focusable = append(c.focusable, c.editor)
}

func (c *Composer) PrevField() {
	c.focusable[c.focused].Focus(false)
	c.focused--
	if c.focused == -1 {
		c.focused = len(c.focusable) - 1
	}
	c.focusable[c.focused].Focus(true)
}

func (c *Composer) NextField() {
	c.focusable[c.focused].Focus(false)
	c.focused = (c.focused + 1) % len(c.focusable)
	c.focusable[c.focused].Focus(true)
}

type headerEditor struct {
	name  string
	input *ui.TextInput
}

func newHeaderEditor(name string, value string) *headerEditor {
	return &headerEditor{
		input: ui.NewTextInput(value),
		name:  name,
	}
}

func (he *headerEditor) Draw(ctx *ui.Context) {
	name := he.name + " "
	size := runewidth.StringWidth(name)
	ctx.Fill(0, 0, size, ctx.Height(), ' ', tcell.StyleDefault)
	ctx.Printf(0, 0, tcell.StyleDefault.Bold(true), "%s", name)
	he.input.Draw(ctx.Subcontext(size, 0, ctx.Width()-size, 1))
}

func (he *headerEditor) Invalidate() {
	he.input.Invalidate()
}

func (he *headerEditor) OnInvalidate(fn func(ui.Drawable)) {
	he.input.OnInvalidate(func(_ ui.Drawable) {
		fn(he)
	})
}

func (he *headerEditor) Focus(focused bool) {
	he.input.Focus(focused)
}

func (he *headerEditor) Event(event tcell.Event) bool {
	return he.input.Event(event)
}

func (he *headerEditor) OnChange(fn func()) {
	he.input.OnChange(func(_ *ui.TextInput) {
		fn()
	})
}

type reviewMessage struct {
	composer *Composer
	grid     *ui.Grid
}

func newReviewMessage(composer *Composer, err error) *reviewMessage {
	grid := ui.NewGrid().Rows([]ui.GridSpec{
		{ui.SIZE_EXACT, 2},
		{ui.SIZE_EXACT, 1},
		{ui.SIZE_WEIGHT, 1},
	}).Columns([]ui.GridSpec{
		{ui.SIZE_WEIGHT, 1},
	})
	if err != nil {
		grid.AddChild(ui.NewText(err.Error()).
			Color(tcell.ColorRed, tcell.ColorDefault))
		grid.AddChild(ui.NewText("Press [q] to close this tab.")).At(1, 0)
	} else {
		// TODO: source this from actual keybindings?
		grid.AddChild(ui.NewText(
			"Send this email? [y]es/[n]o/[e]dit")).At(0, 0)
		grid.AddChild(ui.NewText("Attachments:").
			Reverse(true)).At(1, 0)
		// TODO: Attachments
		grid.AddChild(ui.NewText("(none)")).At(2, 0)
	}

	return &reviewMessage{
		composer: composer,
		grid:     grid,
	}
}

func (rm *reviewMessage) Invalidate() {
	rm.grid.Invalidate()
}

func (rm *reviewMessage) OnInvalidate(fn func(ui.Drawable)) {
	rm.grid.OnInvalidate(func(_ ui.Drawable) {
		fn(rm)
	})
}

func (rm *reviewMessage) Draw(ctx *ui.Context) {
	rm.grid.Draw(ctx)
}
