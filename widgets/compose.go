package widgets

import (
	"bufio"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	gomail "net/mail"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/emersion/go-message"
	"github.com/emersion/go-message/mail"
	"github.com/gdamore/tcell"
	"github.com/mattn/go-runewidth"
	"github.com/pkg/errors"

	"git.sr.ht/~sircmpwn/aerc/config"
	"git.sr.ht/~sircmpwn/aerc/lib/ui"
	"git.sr.ht/~sircmpwn/aerc/worker/types"
)

type Composer struct {
	editors map[string]*headerEditor

	acct   *config.AccountConfig
	config *config.AercConfig

	defaults    map[string]string
	editor      *Terminal
	email       *os.File
	attachments []string
	grid        *ui.Grid
	review      *reviewMessage
	worker      *types.Worker

	focusable []ui.DrawableInteractive
	focused   int
}

func NewComposer(conf *config.AercConfig,
	acct *config.AccountConfig, worker *types.Worker, defaults map[string]string) *Composer {

	if defaults == nil {
		defaults = make(map[string]string)
	}
	if from := defaults["From"]; from == "" {
		defaults["From"] = acct.From
	}

	layout, editors, focusable := buildComposeHeader(conf.Compose.HeaderLayout, defaults)

	header, headerHeight := layout.grid(
		func(header string) ui.Drawable { return editors[header] },
	)

	grid := ui.NewGrid().Rows([]ui.GridSpec{
		{ui.SIZE_EXACT, headerHeight},
		{ui.SIZE_WEIGHT, 1},
	}).Columns([]ui.GridSpec{
		{ui.SIZE_WEIGHT, 1},
	})

	email, err := ioutil.TempFile("", "aerc-compose-*.eml")
	if err != nil {
		// TODO: handle this better
		return nil
	}

	grid.AddChild(header).At(0, 0)

	c := &Composer{
		editors:  editors,
		acct:     acct,
		config:   conf,
		defaults: defaults,
		email:    email,
		grid:     grid,
		worker:   worker,
		// You have to backtab to get to "From", since you usually don't edit it
		focused:   1,
		focusable: focusable,
	}

	c.ShowTerminal()

	return c
}

func buildComposeHeader(layout HeaderLayout, defaults map[string]string) (newLayout HeaderLayout, editors map[string]*headerEditor, focusable []ui.DrawableInteractive) {
	editors = make(map[string]*headerEditor)
	focusable = make([]ui.DrawableInteractive, 0)

	for _, row := range layout {
		for _, h := range row {
			e := newHeaderEditor(h, "")
			editors[h] = e
			switch h {
			case "From":
				// Prepend From to support backtab
				focusable = append([]ui.DrawableInteractive{e}, focusable...)
			default:
				focusable = append(focusable, e)
			}
		}
	}

	// Add Cc/Bcc editors to layout if in defaults and not already visible
	for _, h := range []string{"Cc", "Bcc"} {
		if val, ok := defaults[h]; ok && val != "" {
			if _, ok := editors[h]; !ok {
				e := newHeaderEditor(h, "")
				editors[h] = e
				focusable = append(focusable, e)
				layout = append(layout, []string{h})
			}
		}
	}

	// Set default values for all editors
	for key := range editors {
		if val, ok := defaults[key]; ok {
			editors[key].input.Set(val)
			delete(defaults, key)
		}
	}
	return layout, editors, focusable
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
	c.focused = len(c.editors)
	c.focusable[c.focused].Focus(true)
	return c
}

func (c *Composer) FocusSubject() *Composer {
	c.focusable[c.focused].Focus(false)
	c.focused = 2
	c.focusable[c.focused].Focus(true)
	return c
}

// OnHeaderChange registers an OnChange callback for the specified header.
func (c *Composer) OnHeaderChange(header string, fn func(subject string)) {
	if editor, ok := c.editors[header]; ok {
		editor.OnChange(func() {
			fn(editor.input.String())
		})
	}
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
	if c.editor != nil {
		return c.focusable[c.focused].Event(event)
	}
	return false
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
	if err := c.reloadEmail(); err != nil {
		return nil, nil, err
	}
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
	mhdr.SetText("Message-Id", mail.GenerateMessageID())

	headerKeys := make([]string, 0, len(c.editors))
	for key := range c.editors {
		headerKeys = append(headerKeys, key)
	}
	// Ensure headers which require special processing are included.
	for _, key := range []string{"To", "From", "Cc", "Bcc", "Subject", "Date"} {
		if _, ok := c.editors[key]; !ok {
			headerKeys = append(headerKeys, key)
		}
	}

	for _, h := range headerKeys {
		val := ""
		editor, ok := c.editors[h]
		if ok {
			val = editor.input.String()
		} else {
			val, _ = mhdr.Text(h)
		}
		switch h {
		case "Subject":
			if subject, _ := header.Subject(); subject == "" {
				header.SetSubject(val)
			}
		case "Date":
			if date, err := header.Date(); err != nil || date == (time.Time{}) {
				header.SetDate(time.Now())
			}
		case "From", "To", "Cc", "Bcc": // Address headers
			if val != "" {
				hdrRcpts, err := gomail.ParseAddressList(val)
				if err != nil {
					return nil, nil, errors.Wrapf(err, "ParseAddressList(%s)", val)
				}
				edRcpts := make([]*mail.Address, len(hdrRcpts))
				for i, addr := range hdrRcpts {
					edRcpts[i] = (*mail.Address)(addr)
				}
				header.SetAddressList(h, edRcpts)
				if h != "From" {
					for _, addr := range edRcpts {
						rcpts = append(rcpts, addr.Address)
					}
				}
			}
		default:
			// Handle user configured header editors.
			if ok && !mhdr.Header.Has(h) {
				if val := editor.input.String(); val != "" {
					mhdr.SetText(h, val)
				}
			}
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
	if err := c.reloadEmail(); err != nil {
		return err
	}
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

	if len(c.attachments) == 0 {
		// don't create a multipart email if we only have text
		header.SetContentType("text/plain", map[string]string{"charset": "UTF-8"})
		w, err := mail.CreateSingleInlineWriter(writer, *header)
		if err != nil {
			return errors.Wrap(err, "CreateSingleInlineWriter")
		}
		defer w.Close()

		return writeBody(body, w)
	}

	// otherwise create a multipart email,
	// with a multipart/alternative part for the text
	w, err := mail.CreateWriter(writer, *header)
	if err != nil {
		return errors.Wrap(err, "CreateWriter")
	}
	defer w.Close()

	bh := mail.InlineHeader{}
	bh.SetContentType("text/plain", map[string]string{"charset": "UTF-8"})

	bi, err := w.CreateInline()
	if err != nil {
		return errors.Wrap(err, "CreateInline")
	}
	defer bi.Close()

	bw, err := bi.CreatePart(bh)
	if err != nil {
		return errors.Wrap(err, "CreatePart")
	}
	defer bw.Close()

	if err := writeBody(body, bw); err != nil {
		return err
	}

	for _, a := range c.attachments {
		writeAttachment(a, w)
	}

	return nil
}

func writeBody(body io.Reader, w io.Writer) error {
	if _, err := io.Copy(w, body); err != nil {
		return errors.Wrap(err, "io.Copy")
	}

	return nil
}

// write the attachment specified by path to the message
func writeAttachment(path string, writer *mail.Writer) error {
	filename := filepath.Base(path)

	f, err := os.Open(path)
	if err != nil {
		return errors.Wrap(err, "os.Open")
	}
	defer f.Close()

	reader := bufio.NewReader(f)

	// determine the MIME type
	// http.DetectContentType only cares about the first 512 bytes
	head, err := reader.Peek(512)
	if err != nil {
		return errors.Wrap(err, "Peek")
	}

	mimeString := http.DetectContentType(head)
	// mimeString can contain type and params (like text encoding),
	// so we need to break them apart before passing them to the headers
	mimeType, params, err := mime.ParseMediaType(mimeString)
	if err != nil {
		return errors.Wrap(err, "ParseMediaType")
	}
	params["name"] = filename

	// set header fields
	ah := mail.AttachmentHeader{}
	ah.SetContentType(mimeType, params)
	// setting the filename auto sets the content disposition
	ah.SetFilename(filename)

	aw, err := writer.CreateAttachment(ah)
	if err != nil {
		return errors.Wrap(err, "CreateAttachment")
	}
	defer aw.Close()

	if _, err := reader.WriteTo(aw); err != nil {
		return errors.Wrap(err, "reader.WriteTo")
	}

	return nil
}

func (c *Composer) AddAttachment(path string) {
	c.attachments = append(c.attachments, path)
	if c.review != nil {
		c.grid.RemoveChild(c.review)
		c.review = newReviewMessage(c, nil)
		c.grid.AddChild(c.review).At(1, 0)
	}
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

func (c *Composer) reloadEmail() error {
	name := c.email.Name()
	c.email.Close()
	file, err := os.Open(name)
	if err != nil {
		return errors.Wrap(err, "ReloadEmail")
	}
	c.email = file
	return nil
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
	spec := []ui.GridSpec{{ui.SIZE_EXACT, 2}, {ui.SIZE_EXACT, 1}}
	for range composer.attachments {
		spec = append(spec, ui.GridSpec{ui.SIZE_EXACT, 1})
	}
	// make the last element fill remaining space
	spec = append(spec, ui.GridSpec{ui.SIZE_WEIGHT, 1})

	grid := ui.NewGrid().Rows(spec).Columns([]ui.GridSpec{
		{ui.SIZE_WEIGHT, 1},
	})

	if err != nil {
		grid.AddChild(ui.NewText(err.Error()).
			Color(tcell.ColorRed, tcell.ColorDefault))
		grid.AddChild(ui.NewText("Press [q] to close this tab.")).At(1, 0)
	} else {
		// TODO: source this from actual keybindings?
		grid.AddChild(ui.NewText(
			"Send this email? [y]es/[n]o/[e]dit/[a]ttach")).At(0, 0)
		grid.AddChild(ui.NewText("Attachments:").
			Reverse(true)).At(1, 0)
		if len(composer.attachments) == 0 {
			grid.AddChild(ui.NewText("(none)")).At(2, 0)
		} else {
			for i, a := range composer.attachments {
				grid.AddChild(ui.NewText(a)).At(i+2, 0)
			}
		}
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
