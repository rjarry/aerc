package widgets

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/textproto"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/emersion/go-message/mail"
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"

	"git.sr.ht/~rjarry/aerc/completer"
	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/format"
	"git.sr.ht/~rjarry/aerc/lib/templates"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type Composer struct {
	editors map[string]*headerEditor // indexes in lower case (from / cc / bcc)
	header  *mail.Header
	parent  models.OriginalMail // parent of current message, only set if reply

	acctConfig *config.AccountConfig
	config     *config.AercConfig
	acct       *AccountView
	aerc       *Aerc

	attachments []string
	editor      *Terminal
	email       *os.File
	grid        *ui.Grid
	heditors    *ui.Grid // from, to, cc display a user can jump to
	review      *reviewMessage
	worker      *types.Worker
	completer   *completer.Completer

	layout    HeaderLayout
	focusable []ui.MouseableDrawableInteractive
	focused   int
	sent      bool

	onClose []func(ti *Composer)

	width int
}

func NewComposer(aerc *Aerc, acct *AccountView, conf *config.AercConfig,
	acctConfig *config.AccountConfig, worker *types.Worker, template string,
	h *mail.Header, orig models.OriginalMail) (*Composer, error) {

	if h == nil {
		h = new(mail.Header)
	}
	if fl, err := h.AddressList("from"); err != nil || fl == nil {
		fl, err = mail.ParseAddressList(acctConfig.From)
		// realistically this blows up way before us during the config loading
		if err != nil {
			return nil, err
		}
		if fl != nil {
			h.SetAddressList("from", fl)

		}
	}

	templateData := templates.ParseTemplateData(h, orig)
	cmpl := completer.New(conf.Compose.AddressBookCmd, func(err error) {
		aerc.PushError(
			fmt.Sprintf("could not complete header: %v", err))
		worker.Logger.Printf("could not complete header: %v", err)
	}, aerc.Logger())

	email, err := ioutil.TempFile("", "aerc-compose-*.eml")
	if err != nil {
		// TODO: handle this better
		return nil, err
	}

	c := &Composer{
		acct:       acct,
		acctConfig: acctConfig,
		aerc:       aerc,
		config:     conf,
		header:     h,
		parent:     orig,
		email:      email,
		worker:     worker,
		// You have to backtab to get to "From", since you usually don't edit it
		focused:   1,
		completer: cmpl,
	}
	c.buildComposeHeader(aerc, cmpl)

	if err := c.AddTemplate(template, templateData); err != nil {
		return nil, err
	}
	c.AddSignature()

	c.updateGrid()
	c.ShowTerminal()

	return c, nil
}

func (c *Composer) buildComposeHeader(aerc *Aerc, cmpl *completer.Completer) {

	c.layout = aerc.conf.Compose.HeaderLayout
	c.editors = make(map[string]*headerEditor)
	c.focusable = make([]ui.MouseableDrawableInteractive, 0)

	for i, row := range c.layout {
		for j, h := range row {
			h = strings.ToLower(h)
			c.layout[i][j] = h // normalize to lowercase
			e := newHeaderEditor(h, c.header, aerc.SelectedAccount().UiConfig())
			if aerc.conf.Ui.CompletionPopovers {
				e.input.TabComplete(cmpl.ForHeader(h),
					aerc.SelectedAccount().UiConfig().CompletionDelay)
			}
			c.editors[h] = e
			switch h {
			case "from":
				// Prepend From to support backtab
				c.focusable = append([]ui.MouseableDrawableInteractive{e}, c.focusable...)
			default:
				c.focusable = append(c.focusable, e)
			}
		}
	}

	// Add Cc/Bcc editors to layout if present in header and not already visible
	for _, h := range []string{"cc", "bcc"} {
		if c.header.Has(h) {
			if _, ok := c.editors[h]; !ok {
				e := newHeaderEditor(h, c.header, aerc.SelectedAccount().UiConfig())
				if aerc.conf.Ui.CompletionPopovers {
					e.input.TabComplete(cmpl.ForHeader(h), aerc.SelectedAccount().UiConfig().CompletionDelay)
				}
				c.editors[h] = e
				c.focusable = append(c.focusable, e)
				c.layout = append(c.layout, []string{h})
			}
		}
	}

	// load current header values into all editors
	for _, e := range c.editors {
		e.loadValue()
	}
}

func (c *Composer) SetSent() {
	c.sent = true
}

func (c *Composer) Sent() bool {
	return c.sent
}

// Note: this does not reload the editor. You must call this before the first
// Draw() call.
func (c *Composer) SetContents(reader io.Reader) *Composer {
	c.email.Seek(0, io.SeekStart)
	io.Copy(c.email, reader)
	c.email.Sync()
	c.email.Seek(0, io.SeekStart)
	return c
}

func (c *Composer) AppendContents(reader io.Reader) {
	c.email.Seek(0, io.SeekEnd)
	io.Copy(c.email, reader)
	c.email.Sync()
}

func (c *Composer) AddTemplate(template string, data interface{}) error {
	if template == "" {
		return nil
	}

	templateText, err := templates.ParseTemplateFromFile(
		template, c.config.Templates.TemplateDirs, data)
	if err != nil {
		return err
	}

	mr, err := mail.CreateReader(templateText)
	if err != nil {
		return fmt.Errorf("Template loading failed: %v", err)
	}

	// copy the headers contained in the template to the compose headers
	hf := mr.Header.Fields()
	for hf.Next() {
		c.header.Set(hf.Key(), hf.Value())
	}

	part, err := mr.NextPart()
	if err != nil {
		return fmt.Errorf("Could not get body of template: %v", err)
	}

	c.AppendContents(part.Body)
	return nil
}

func (c *Composer) AddSignature() {
	var signature []byte
	if c.acctConfig.SignatureCmd != "" {
		var err error
		signature, err = c.readSignatureFromCmd()
		if err != nil {
			signature = c.readSignatureFromFile()
		}
	} else {
		signature = c.readSignatureFromFile()
	}
	c.AppendContents(bytes.NewReader(signature))
}

func (c *Composer) readSignatureFromCmd() ([]byte, error) {
	sigCmd := c.acctConfig.SignatureCmd
	cmd := exec.Command("sh", "-c", sigCmd)
	signature, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return signature, nil
}

func (c *Composer) readSignatureFromFile() []byte {
	sigFile := c.acctConfig.SignatureFile
	if sigFile == "" {
		return nil
	}
	sigFile, err := homedir.Expand(sigFile)
	if err != nil {
		return nil
	}
	signature, err := ioutil.ReadFile(sigFile)
	if err != nil {
		c.aerc.PushError(
			fmt.Sprintf(" Error loading signature from file: %v", sigFile))
		return nil
	}
	return signature
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

func (c *Composer) FocusRecipient() *Composer {
	c.focusable[c.focused].Focus(false)
	c.focused = 1
	c.focusable[c.focused].Focus(true)
	return c
}

// OnHeaderChange registers an OnChange callback for the specified header.
func (c *Composer) OnHeaderChange(header string, fn func(subject string)) {
	if editor, ok := c.editors[strings.ToLower(header)]; ok {
		editor.OnChange(func() {
			fn(editor.input.String())
		})
	}
}

func (c *Composer) OnClose(fn func(composer *Composer)) {
	c.onClose = append(c.onClose, fn)
}

func (c *Composer) Draw(ctx *ui.Context) {
	c.width = ctx.Width()
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
	for _, onClose := range c.onClose {
		onClose(c)
	}
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

func (c *Composer) MouseEvent(localX int, localY int, event tcell.Event) {
	c.grid.MouseEvent(localX, localY, event)
	for _, e := range c.focusable {
		he, ok := e.(*headerEditor)
		if ok && he.focused {
			c.FocusEditor(he)
		}
	}
}

func (c *Composer) Focus(focus bool) {
	c.focusable[c.focused].Focus(focus)
}

func (c *Composer) Config() *config.AccountConfig {
	return c.acctConfig
}

func (c *Composer) Account() *AccountView {
	return c.acct
}

func (c *Composer) Worker() *types.Worker {
	return c.worker
}

//PrepareHeader finalizes the header, adding the value from the editors
func (c *Composer) PrepareHeader() (*mail.Header, error) {
	for _, editor := range c.editors {
		editor.storeValue()
	}

	// control headers not normally set by the user
	// repeated calls to PrepareHeader should be a noop
	if !c.header.Has("Message-Id") {
		err := c.header.GenerateMessageID()
		if err != nil {
			return nil, err
		}
	}
	if !c.header.Has("Date") {
		c.header.SetDate(time.Now())
	}
	return c.header, nil
}

func (c *Composer) WriteMessage(header *mail.Header, writer io.Writer) error {
	if err := c.reloadEmail(); err != nil {
		return err
	}

	if len(c.attachments) == 0 {
		// don't create a multipart email if we only have text
		return writeInlineBody(header, c.email, writer)
	}

	// otherwise create a multipart email,
	// with a multipart/alternative part for the text
	w, err := mail.CreateWriter(writer, *header)
	if err != nil {
		return errors.Wrap(err, "CreateWriter")
	}
	defer w.Close()

	if err := writeMultipartBody(c.email, w); err != nil {
		return errors.Wrap(err, "writeMultipartBody")
	}

	for _, a := range c.attachments {
		if err := writeAttachment(a, w); err != nil {
			return errors.Wrap(err, "writeAttachment")
		}
	}

	return nil
}

func writeInlineBody(header *mail.Header, body io.Reader, writer io.Writer) error {
	header.SetContentType("text/plain", map[string]string{"charset": "UTF-8"})
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

// write the message body to the multipart message
func writeMultipartBody(body io.Reader, w *mail.Writer) error {
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
	if _, err := io.Copy(bw, body); err != nil {
		return errors.Wrap(err, "io.Copy")
	}
	return nil
}

// write the attachment specified by path to the message
func writeAttachment(path string, writer *mail.Writer) error {
	f, err := os.Open(path)
	if err != nil {
		return errors.Wrap(err, "os.Open")
	}
	defer f.Close()

	reader := bufio.NewReader(f)

	// if we have an extension, prefer that instead of trying to sniff the header.
	// That's generally more accurate than sniffing as lots of things are zip files
	// under the hood, e.g. most office file types
	ext := filepath.Ext(path)
	var mimeString string
	if mimeString = mime.TypeByExtension(ext); mimeString != "" {
		// found it in the DB
	} else {
		// Sniff the mime type instead
		// http.DetectContentType only cares about the first 512 bytes
		head, err := reader.Peek(512)
		if err != nil && err != io.EOF {
			return errors.Wrap(err, "Peek")
		}
		mimeString = http.DetectContentType(head)
	}

	// mimeString can contain type and params (like text encoding),
	// so we need to break them apart before passing them to the headers
	mimeType, params, err := mime.ParseMediaType(mimeString)
	if err != nil {
		return errors.Wrap(err, "ParseMediaType")
	}
	filename := filepath.Base(path)
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

func (c *Composer) GetAttachments() []string {
	return c.attachments
}

func (c *Composer) AddAttachment(path string) {
	c.attachments = append(c.attachments, path)
	c.resetReview()
}

func (c *Composer) DeleteAttachment(path string) error {
	for i, a := range c.attachments {
		if a == path {
			c.attachments = append(c.attachments[:i], c.attachments[i+1:]...)
			c.resetReview()
			return nil
		}
	}

	return errors.New("attachment does not exist")
}

func (c *Composer) resetReview() {
	if c.review != nil {
		c.grid.RemoveChild(c.review)
		c.review = newReviewMessage(c, nil)
		c.grid.AddChild(c.review).At(2, 0)
	}
}

func (c *Composer) termEvent(event tcell.Event) bool {
	switch event := event.(type) {
	case *tcell.EventMouse:
		switch event.Buttons() {
		case tcell.Button1:
			c.FocusTerminal()
			return true
		}
	}
	return false
}

func (c *Composer) termClosed(err error) {
	c.grid.RemoveChild(c.editor)
	c.review = newReviewMessage(c, err)
	c.grid.AddChild(c.review).At(2, 0)
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
	c.editor.OnEvent = c.termEvent
	c.editor.OnClose = c.termClosed
	c.grid.AddChild(c.editor).At(2, 0)
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

func (c *Composer) FocusEditor(editor *headerEditor) {
	c.focusable[c.focused].Focus(false)
	for i, e := range c.focusable {
		if e == editor {
			c.focused = i
			break
		}
	}
	c.focusable[c.focused].Focus(true)
}

// AddEditor appends a new header editor to the compose window.
func (c *Composer) AddEditor(header string, value string, appendHeader bool) {
	var editor *headerEditor
	header = strings.ToLower(header)
	if e, ok := c.editors[header]; ok {
		e.storeValue() // flush modifications from the user to the header
		editor = e
	} else {
		e := newHeaderEditor(header, c.header,
			c.aerc.SelectedAccount().UiConfig())
		if c.config.Ui.CompletionPopovers {
			e.input.TabComplete(c.completer.ForHeader(header),
				c.config.Ui.CompletionDelay)
		}
		c.editors[header] = e
		c.layout = append(c.layout, []string{header})
		// Insert focus of new editor before terminal editor
		c.focusable = append(
			c.focusable[:len(c.focusable)-1],
			e,
			c.focusable[len(c.focusable)-1],
		)
		editor = e
	}

	if appendHeader {
		currVal := editor.input.String()
		if currVal != "" {
			value = strings.TrimSpace(currVal) + ", " + value
		}
	}
	if value != "" || appendHeader {
		c.editors[header].input.Set(value)
		editor.storeValue()
	}
	if value == "" {
		c.FocusEditor(c.editors[header])
	}
	c.updateGrid()
}

// updateGrid should be called when the underlying header layout is changed.
func (c *Composer) updateGrid() {
	heditors, height := c.layout.grid(
		func(h string) ui.Drawable {
			return c.editors[h]
		},
	)

	if c.grid == nil {
		c.grid = ui.NewGrid().Columns([]ui.GridSpec{
			{ui.SIZE_WEIGHT, ui.Const(1)},
		})
	}

	c.grid.Rows([]ui.GridSpec{
		{ui.SIZE_EXACT, ui.Const(height)},
		{ui.SIZE_EXACT, ui.Const(1)},
		{ui.SIZE_WEIGHT, ui.Const(1)},
	})

	if c.heditors != nil {
		c.grid.RemoveChild(c.heditors)
	}
	borderStyle := c.config.Ui.GetStyle(config.STYLE_BORDER)
	c.heditors = heditors
	c.grid.AddChild(c.heditors).At(0, 0)
	c.grid.AddChild(ui.NewFill(' ', borderStyle)).At(1, 0)
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
	name     string
	header   *mail.Header
	focused  bool
	input    *ui.TextInput
	uiConfig config.UIConfig
}

func newHeaderEditor(name string, h *mail.Header,
	uiConfig config.UIConfig) *headerEditor {
	he := &headerEditor{
		input:    ui.NewTextInput("", uiConfig),
		name:     name,
		header:   h,
		uiConfig: uiConfig,
	}
	he.loadValue()
	return he
}

//extractHumanHeaderValue extracts the human readable string for key from the
//header. If a parsing error occurs the raw value is returned
func extractHumanHeaderValue(key string, h *mail.Header) string {
	var val string
	var err error
	switch strings.ToLower(key) {
	case "to", "from", "cc", "bcc":
		var list []*mail.Address
		list, err = h.AddressList(key)
		val = format.FormatAddresses(list)
	default:
		val, err = h.Text(key)
	}
	if err != nil {
		// if we can't parse it, show it raw
		val = h.Get(key)
	}
	return val
}

//loadValue loads the value of he.name form the underlying header
//the value is decoded and meant for human consumption.
//decoding issues are ignored and return their raw values
func (he *headerEditor) loadValue() {
	he.input.Set(extractHumanHeaderValue(he.name, he.header))
	he.input.Invalidate()
}

//storeValue writes the current state back to the underlying header.
//errors are ignored
func (he *headerEditor) storeValue() {
	val := he.input.String()
	switch strings.ToLower(he.name) {
	case "to", "from", "cc", "bcc":
		list, err := mail.ParseAddressList(val)
		if err == nil {
			he.header.SetAddressList(he.name, list)
		} else {
			// garbage, but it'll blow up upon sending and the user can
			// fix the issue
			he.header.SetText(he.name, val)
		}
		val = format.FormatAddresses(list)
	default:
		he.header.SetText(he.name, val)
	}
}

//setValue overwrites the current value of the header editor and flushes it
//to the underlying header
func (he *headerEditor) setValue(val string) {
	he.input.Set(val)
	he.storeValue()
}

func (he *headerEditor) Draw(ctx *ui.Context) {
	name := textproto.CanonicalMIMEHeaderKey(he.name)
	// Extra character to put a blank cell between the header and the input
	size := runewidth.StringWidth(name) + 1
	defaultStyle := he.uiConfig.GetStyle(config.STYLE_DEFAULT)
	headerStyle := he.uiConfig.GetStyle(config.STYLE_HEADER)
	ctx.Fill(0, 0, size, ctx.Height(), ' ', defaultStyle)
	ctx.Printf(0, 0, headerStyle, "%s", name)
	he.input.Draw(ctx.Subcontext(size, 0, ctx.Width()-size, 1))
}

func (he *headerEditor) MouseEvent(localX int, localY int, event tcell.Event) {
	switch event := event.(type) {
	case *tcell.EventMouse:
		switch event.Buttons() {
		case tcell.Button1:
			he.focused = true
		}

		width := runewidth.StringWidth(he.name + " ")
		if localX >= width {
			he.input.MouseEvent(localX-width, localY, event)
		}
	}
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
	he.focused = focused
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
	spec := []ui.GridSpec{
		{ui.SIZE_EXACT, ui.Const(2)},
		{ui.SIZE_EXACT, ui.Const(1)},
	}
	for i := 0; i < len(composer.attachments)-1; i++ {
		spec = append(spec, ui.GridSpec{ui.SIZE_EXACT, ui.Const(1)})
	}
	// make the last element fill remaining space
	spec = append(spec, ui.GridSpec{ui.SIZE_WEIGHT, ui.Const(1)})

	grid := ui.NewGrid().Rows(spec).Columns([]ui.GridSpec{
		{ui.SIZE_WEIGHT, ui.Const(1)},
	})

	uiConfig := composer.config.Ui

	if err != nil {
		grid.AddChild(ui.NewText(err.Error(), uiConfig.GetStyle(config.STYLE_ERROR)))
		grid.AddChild(ui.NewText("Press [q] to close this tab.",
			uiConfig.GetStyle(config.STYLE_DEFAULT))).At(1, 0)
	} else {
		// TODO: source this from actual keybindings?
		grid.AddChild(ui.NewText("Send this email? [y]es/[n]o/[e]dit/[a]ttach",
			uiConfig.GetStyle(config.STYLE_DEFAULT))).At(0, 0)
		grid.AddChild(ui.NewText("Attachments:",
			uiConfig.GetStyle(config.STYLE_TITLE))).At(1, 0)
		if len(composer.attachments) == 0 {
			grid.AddChild(ui.NewText("(none)",
				uiConfig.GetStyle(config.STYLE_DEFAULT))).At(2, 0)
		} else {
			for i, a := range composer.attachments {
				grid.AddChild(ui.NewText(a, uiConfig.GetStyle(config.STYLE_DEFAULT))).
					At(i+2, 0)
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
