package widgets

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/textproto"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/emersion/go-message/mail"
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/pkg/errors"

	"git.sr.ht/~rjarry/aerc/completer"
	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/format"
	"git.sr.ht/~rjarry/aerc/lib/state"
	"git.sr.ht/~rjarry/aerc/lib/templates"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/lib/xdg"
	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type Composer struct {
	sync.Mutex
	editors map[string]*headerEditor // indexes in lower case (from / cc / bcc)
	header  *mail.Header
	parent  *models.OriginalMail // parent of current message, only set if reply

	acctConfig *config.AccountConfig
	acct       *AccountView
	aerc       *Aerc

	attachments []lib.Attachment
	editor      *Terminal
	email       *os.File
	grid        atomic.Value
	heditors    atomic.Value // from, to, cc display a user can jump to
	review      *reviewMessage
	worker      *types.Worker
	completer   *completer.Completer
	crypto      *cryptoStatus
	sign        bool
	encrypt     bool
	attachKey   bool
	editHeaders bool

	layout    HeaderLayout
	focusable []ui.MouseableDrawableInteractive
	focused   int
	sent      bool
	archive   string

	recalledFrom string
	postponed    bool

	onClose []func(ti *Composer)

	width int

	textParts []*lib.Part
	Tab       *ui.Tab
}

func NewComposer(
	aerc *Aerc, acct *AccountView, acctConfig *config.AccountConfig,
	worker *types.Worker, editHeaders bool, template string,
	h *mail.Header, orig *models.OriginalMail, body io.Reader,
) (*Composer, error) {
	if h == nil {
		h = new(mail.Header)
	}

	email, err := os.CreateTemp("", "aerc-compose-*.eml")
	if err != nil {
		// TODO: handle this better
		return nil, err
	}

	c := &Composer{
		acct:       acct,
		acctConfig: acctConfig,
		aerc:       aerc,
		header:     h,
		parent:     orig,
		email:      email,
		worker:     worker,
		// You have to backtab to get to "From", since you usually don't edit it
		focused:   1,
		completer: nil,

		editHeaders: editHeaders,
	}

	data := state.NewDataSetter()
	data.SetAccount(acct.acct)
	data.SetFolder(acct.Directories().SelectedDirectory())
	data.SetHeaders(h, orig)
	data.SetComposer(c)
	if err := c.addTemplate(template, data.Data(), body); err != nil {
		return nil, err
	}
	if sig, err := c.HasSignature(); !sig && err == nil {
		c.AddSignature()
	} else if err != nil {
		return nil, err
	}
	if err := c.setupFor(acct); err != nil {
		return nil, err
	}

	if err := c.ShowTerminal(editHeaders); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Composer) SwitchAccount(newAcct *AccountView) error {
	if c.acct == newAcct {
		log.Tracef("same accounts: no switch")
		return nil
	}
	// sync the header with the editors
	for _, editor := range c.editors {
		editor.storeValue()
	}
	// ensure that from header is updated, so remove it
	c.header.Del("from")
	c.header.Del("message-id")
	// update entire composer with new the account
	if err := c.setupFor(newAcct); err != nil {
		return err
	}
	// sync the header with the editors
	for _, editor := range c.editors {
		editor.loadValue()
	}
	c.resetReview()
	c.Invalidate()
	log.Debugf("account successfully switched")
	return nil
}

func (c *Composer) setupFor(view *AccountView) error {
	c.Lock()
	defer c.Unlock()
	// set new account
	c.acct = view
	c.worker = view.Worker()
	c.acctConfig = c.acct.AccountConfig()
	// Set from header if not already in header
	if fl, err := c.header.AddressList("from"); err != nil || fl == nil {
		c.header.SetAddressList("from", []*mail.Address{view.acct.From})
	}
	if !c.header.Has("to") {
		c.header.SetAddressList("to", make([]*mail.Address, 0))
	}
	if !c.header.Has("subject") {
		c.header.SetSubject("")
	}

	// update completer
	cmd := view.acct.AddressBookCmd
	if cmd == "" {
		cmd = config.Compose.AddressBookCmd
	}
	cmpl := completer.New(cmd, func(err error) {
		c.aerc.PushError(
			fmt.Sprintf("could not complete header: %v", err))
		log.Errorf("could not complete header: %v", err)
	})
	c.completer = cmpl

	// if editor already exists, we have to get it from the focusable slice
	// because this will be rebuild during buildComposeHeader()
	var focusEditor ui.MouseableDrawableInteractive
	if c.editor != nil && len(c.focusable) > 0 {
		focusEditor = c.focusable[len(c.focusable)-1]
	}

	// rebuild editors and focusable slice
	c.buildComposeHeader(c.aerc, cmpl)

	// restore the editor in the focusable list
	if focusEditor != nil {
		c.focusable = append(c.focusable, focusEditor)
	}
	if c.focused >= len(c.focusable) {
		c.focused = len(c.focusable) - 1
	}

	// update the crypto parts
	c.crypto = nil
	c.sign = false
	if c.acct.acct.PgpAutoSign {
		err := c.SetSign(true)
		log.Warnf("failed to enable message signing: %v", err)
	}
	c.encrypt = false
	if c.acct.acct.PgpOpportunisticEncrypt {
		c.SetEncrypt(true)
	}
	err := c.updateCrypto()
	if err != nil {
		log.Warnf("failed to update crypto: %v", err)
	}

	// redraw the grid
	c.updateGrid()

	return nil
}

func (c *Composer) buildComposeHeader(aerc *Aerc, cmpl *completer.Completer) {
	c.layout = config.Compose.HeaderLayout
	c.editors = make(map[string]*headerEditor)
	c.focusable = make([]ui.MouseableDrawableInteractive, 0)
	uiConfig := c.acct.UiConfig()

	for i, row := range c.layout {
		for j, h := range row {
			h = strings.ToLower(h)
			c.layout[i][j] = h // normalize to lowercase
			e := newHeaderEditor(h, c.header, uiConfig)
			if uiConfig.CompletionPopovers {
				e.input.TabComplete(
					cmpl.ForHeader(h),
					uiConfig.CompletionDelay,
					uiConfig.CompletionMinChars,
				)
			}
			c.editors[h] = e
			switch h {
			case "from":
				// Prepend From to support backtab
				c.focusable = append([]ui.MouseableDrawableInteractive{e}, c.focusable...)
			default:
				c.focusable = append(c.focusable, e)
			}
			e.OnChange(func() {
				c.setTitle()
				ui.Invalidate()
			})
			e.OnFocusLost(func() {
				c.PrepareHeader() //nolint:errcheck // tab title only, fine if it's not valid yet
				c.setTitle()
				ui.Invalidate()
			})
		}
	}

	// Add Cc/Bcc editors to layout if present in header and not already visible
	for _, h := range []string{"cc", "bcc"} {
		if c.header.Has(h) {
			if _, ok := c.editors[h]; !ok {
				e := newHeaderEditor(h, c.header, uiConfig)
				if uiConfig.CompletionPopovers {
					e.input.TabComplete(
						cmpl.ForHeader(h),
						uiConfig.CompletionDelay,
						uiConfig.CompletionMinChars,
					)
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

func (c *Composer) headerOrder() []string {
	var order []string
	for _, row := range c.layout {
		order = append(order, row...)
	}
	return order
}

func (c *Composer) SetSent(archive string) {
	c.sent = true
	c.archive = archive
}

func (c *Composer) Sent() bool {
	return c.sent
}

func (c *Composer) SetPostponed() {
	c.postponed = true
}

func (c *Composer) Postponed() bool {
	return c.postponed
}

func (c *Composer) SetRecalledFrom(folder string) {
	c.recalledFrom = folder
}

func (c *Composer) RecalledFrom() string {
	return c.recalledFrom
}

func (c *Composer) Archive() string {
	return c.archive
}

func (c *Composer) SetAttachKey(attach bool) error {
	if !attach {
		name := c.crypto.signKey + ".asc"
		found := false
		for _, a := range c.attachments {
			if a.Name() == name {
				found = true
			}
		}
		if found {
			err := c.DeleteAttachment(name)
			if err != nil {
				return fmt.Errorf("failed to delete attachment '%s: %w", name, err)
			}
		} else {
			attach = !attach
		}
	}
	if attach {
		var s string
		var err error
		if c.crypto.signKey == "" {
			if c.acctConfig.PgpKeyId != "" {
				s = c.acctConfig.PgpKeyId
			} else {
				s = c.acctConfig.From.Address
			}
			c.crypto.signKey, err = c.aerc.Crypto.GetSignerKeyId(s)
			if err != nil {
				return err
			}
		}

		r, err := c.aerc.Crypto.ExportKey(c.crypto.signKey)
		if err != nil {
			return err
		}

		newPart, err := lib.NewPart(
			"application/pgp-keys",
			map[string]string{"charset": "UTF-8"},
			r,
		)
		if err != nil {
			return err
		}
		c.attachments = append(c.attachments,
			lib.NewPartAttachment(
				newPart,
				c.crypto.signKey+".asc",
			),
		)

	}

	c.attachKey = attach

	c.resetReview()
	return nil
}

func (c *Composer) AttachKey() bool {
	return c.attachKey
}

func (c *Composer) SetSign(sign bool) error {
	c.sign = sign
	err := c.updateCrypto()
	if err != nil {
		c.sign = !sign
		return fmt.Errorf("Cannot sign message: %w", err)
	}
	return nil
}

func (c *Composer) Sign() bool {
	return c.sign
}

func (c *Composer) SetEncrypt(encrypt bool) *Composer {
	if !encrypt {
		c.encrypt = encrypt
		err := c.updateCrypto()
		if err != nil {
			log.Warnf("failed to update crypto: %v", err)
		}
		return c
	}
	// Check on any attempt to encrypt, and any lost focus of "to", "cc", or
	// "bcc" field. Use OnFocusLost instead of OnChange to limit keyring checks
	c.encrypt = c.checkEncryptionKeys("")
	if c.crypto.setEncOneShot {
		// Prevent registering a lot of callbacks
		c.OnFocusLost("to", c.checkEncryptionKeys)
		c.OnFocusLost("cc", c.checkEncryptionKeys)
		c.OnFocusLost("bcc", c.checkEncryptionKeys)
		c.crypto.setEncOneShot = false
	}
	return c
}

func (c *Composer) Encrypt() bool {
	return c.encrypt
}

func (c *Composer) updateCrypto() error {
	if c.crypto == nil {
		uiConfig := c.acct.UiConfig()
		c.crypto = newCryptoStatus(uiConfig)
	}
	if c.sign {
		cp := c.aerc.Crypto
		s, err := c.Signer()
		if err != nil {
			return errors.Wrap(err, "Signer")
		}
		c.crypto.signKey, err = cp.GetSignerKeyId(s)
		if err != nil {
			return err
		}
	}

	st := ""
	switch {
	case c.sign && c.encrypt:
		st = fmt.Sprintf("Sign (%s) & Encrypt", c.crypto.signKey)
	case c.sign:
		st = fmt.Sprintf("Sign (%s)", c.crypto.signKey)
	case c.encrypt:
		st = "Encrypt"
	}
	c.crypto.status.Text(st)

	c.updateGrid()

	return nil
}

func (c *Composer) writeEml(reader io.Reader) error {
	// .eml files must always use '\r\n' line endings, but some editors
	// don't support these, so if they are using one of those, the
	// line-endings are transformed
	lineEnding := "\r\n"
	if config.Compose.LFEditor {
		lineEnding = "\n"
	}

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		_, err := c.email.WriteString(scanner.Text() + lineEnding)
		if err != nil {
			return err
		}
	}
	if scanner.Err() != nil {
		return scanner.Err()
	}
	return c.email.Sync()
}

// Note: this does not reload the editor. You must call this before the first
// Draw() call.
func (c *Composer) setContents(reader io.Reader) error {
	_, err := c.email.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	err = c.email.Truncate(0)
	if err != nil {
		return err
	}
	lineEnding := "\r\n"
	if config.Compose.LFEditor {
		lineEnding = "\n"
	}

	if c.editHeaders {
		for _, h := range c.headerOrder() {
			var value string
			switch h {
			case "to", "from", "cc", "bcc":
				addresses, err := c.header.AddressList(h)
				if err != nil {
					log.Warnf("header.AddressList: %s", err)
					value, err = c.header.Text(h)
					if err != nil {
						log.Warnf("header.Text: %s", err)
						value = c.header.Get(h)
					}
				} else {
					addr := make([]string, 0, len(addresses))
					for _, a := range addresses {
						addr = append(addr, format.AddressForHumans(a))
					}
					value = strings.Join(addr, ","+lineEnding+"\t")
				}
			default:
				value, err = c.header.Text(h)
				if err != nil {
					log.Warnf("header.Text: %s", err)
					value = c.header.Get(h)
				}
			}
			key := textproto.CanonicalMIMEHeaderKey(h)
			_, err = fmt.Fprintf(c.email, "%s: %s"+lineEnding, key, value)
			if err != nil {
				return err
			}
		}
		_, err = c.email.WriteString(lineEnding)
		if err != nil {
			return err
		}
	}
	return c.writeEml(reader)
}

func (c *Composer) appendContents(reader io.Reader) error {
	_, err := c.email.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	return c.writeEml(reader)
}

func (c *Composer) AppendPart(mimetype string, params map[string]string, body io.Reader) error {
	if !strings.HasPrefix(mimetype, "text") {
		return fmt.Errorf("can only append text mimetypes")
	}
	for _, part := range c.textParts {
		if part.MimeType == mimetype {
			return fmt.Errorf("%s part already exists", mimetype)
		}
	}
	newPart, err := lib.NewPart(mimetype, params, body)
	if err != nil {
		return err
	}
	c.textParts = append(c.textParts, newPart)
	c.resetReview()
	return nil
}

func (c *Composer) RemovePart(mimetype string) error {
	if mimetype == "text/plain" {
		return fmt.Errorf("cannot remove text/plain parts")
	}
	for i, part := range c.textParts {
		if part.MimeType != mimetype {
			continue
		}
		c.textParts = append(c.textParts[:i], c.textParts[i+1:]...)
		c.resetReview()
		return nil
	}
	return fmt.Errorf("%s part not found", mimetype)
}

func (c *Composer) addTemplate(
	template string, data models.TemplateData, body io.Reader,
) error {
	var readers []io.Reader

	if template != "" {
		templateText, err := templates.ParseTemplateFromFile(
			template, config.Templates.TemplateDirs, data)
		if err != nil {
			return err
		}
		readers = append(readers, templateText)
	}
	if body != nil {
		if len(readers) == 0 {
			readers = append(readers, bytes.NewReader([]byte("\r\n")))
		}
		readers = append(readers, body)
	}
	if len(readers) == 0 {
		return nil
	}

	buf, err := io.ReadAll(io.MultiReader(readers...))
	if err != nil {
		return err
	}

	mr, err := mail.CreateReader(bytes.NewReader(buf))
	if err != nil {
		// no headers in the template nor body
		return c.setContents(bytes.NewReader(buf))
	}

	// copy the headers contained in the template to the compose headers
	hf := mr.Header.Fields()
	for hf.Next() {
		c.header.Set(hf.Key(), hf.Value())
	}

	part, err := mr.NextPart()
	if err != nil {
		return fmt.Errorf("NextPart: %w", err)
	}

	return c.setContents(part.Body)
}

func (c *Composer) HasSignature() (bool, error) {
	buf, err := c.GetBody()
	if err != nil {
		return false, err
	}
	found := false
	scanner := bufio.NewScanner(buf)
	for scanner.Scan() {
		if scanner.Text() == "-- " {
			found = true
			break
		}
	}
	return found, scanner.Err()
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
	if len(bytes.TrimSpace(signature)) == 0 {
		return
	}
	signature = ensureSignatureDelimiter(signature)
	err := c.appendContents(bytes.NewReader(signature))
	if err != nil {
		log.Errorf("appendContents: %s", err)
	}
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
	sigFile = xdg.ExpandHome(sigFile)
	signature, err := os.ReadFile(sigFile)
	if err != nil {
		c.aerc.PushError(
			fmt.Sprintf(" Error loading signature from file: %v", sigFile))
		return nil
	}
	return signature
}

func ensureSignatureDelimiter(signature []byte) []byte {
	buf := bytes.NewBuffer(signature)
	scanner := bufio.NewScanner(buf)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "-- " {
			// signature contains standard delimiter, we're good
			return signature
		}
	}
	// signature does not contain standard delimiter, prepend one
	sig := "\n\n-- \n" + strings.TrimLeft(string(signature), " \t\r\n")
	return []byte(sig)
}

func (c *Composer) GetBody() (*bytes.Buffer, error) {
	_, err := c.email.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(c.email)
	if c.editHeaders {
		// skip headers
		for scanner.Scan() {
			if scanner.Text() == "" {
				break // stop on first empty line
			}
		}
	}
	// .eml files must always use '\r\n' line endings
	buf := new(bytes.Buffer)
	for scanner.Scan() {
		buf.WriteString(scanner.Text() + "\r\n")
	}
	err = scanner.Err()
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func (c *Composer) FocusTerminal() *Composer {
	c.Lock()
	defer c.Unlock()
	return c.focusTerminalPriv()
}

func (c *Composer) focusTerminalPriv() *Composer {
	if c.editor == nil {
		return c
	}
	c.focusActiveWidget(false)
	c.focused = len(c.focusable) - 1
	c.focusActiveWidget(true)
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

// OnFocusLost registers an OnFocusLost callback for the specified header.
func (c *Composer) OnFocusLost(header string, fn func(input string) bool) {
	if editor, ok := c.editors[strings.ToLower(header)]; ok {
		editor.OnFocusLost(func() {
			fn(editor.input.String())
		})
	}
}

func (c *Composer) OnClose(fn func(composer *Composer)) {
	c.onClose = append(c.onClose, fn)
}

func (c *Composer) Draw(ctx *ui.Context) {
	c.setTitle()
	c.width = ctx.Width()
	c.grid.Load().(*ui.Grid).Draw(ctx)
}

func (c *Composer) Invalidate() {
	ui.Invalidate()
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
	c.Lock()
	defer c.Unlock()
	switch c.editor {
	case nil:
		return "compose::review"
	case c.focusedWidget():
		return "compose::editor"
	default:
		return "compose"
	}
}

func (c *Composer) focusedWidget() ui.MouseableDrawableInteractive {
	if c.focused < 0 || c.focused >= len(c.focusable) {
		return nil
	}
	return c.focusable[c.focused]
}

func (c *Composer) focusActiveWidget(focus bool) {
	if w := c.focusedWidget(); w != nil {
		w.Focus(focus)
	}
}

func (c *Composer) Event(event tcell.Event) bool {
	c.Lock()
	defer c.Unlock()
	if w := c.focusedWidget(); c.editor != nil && w != nil {
		return w.Event(event)
	}
	return false
}

func (c *Composer) MouseEvent(localX int, localY int, event tcell.Event) {
	c.Lock()
	for _, e := range c.focusable {
		he, ok := e.(*headerEditor)
		if ok && he.focused {
			he.focused = false
		}
	}
	c.Unlock()
	c.grid.Load().(*ui.Grid).MouseEvent(localX, localY, event)
	c.Lock()
	defer c.Unlock()
	for i, e := range c.focusable {
		he, ok := e.(*headerEditor)
		if ok && he.focused {
			c.focusActiveWidget(false)
			c.focused = i
			c.focusActiveWidget(true)
			return
		}
	}
}

func (c *Composer) Focus(focus bool) {
	c.Lock()
	c.focusActiveWidget(focus)
	c.Unlock()
}

func (c *Composer) Show(visible bool) {
	c.Lock()
	if w := c.focusedWidget(); w != nil {
		if vis, ok := w.(ui.Visible); ok {
			vis.Show(visible)
		}
	}
	c.Unlock()
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

// PrepareHeader finalizes the header, adding the value from the editors
func (c *Composer) PrepareHeader() (*mail.Header, error) {
	for _, editor := range c.editors {
		editor.storeValue()
	}

	// control headers not normally set by the user
	// repeated calls to PrepareHeader should be a noop
	if !c.header.Has("Message-Id") {
		hostname, err := getMessageIdHostname(c)
		if err != nil {
			return nil, err
		}
		if err := c.header.GenerateMessageIDWithHostname(hostname); err != nil {
			return nil, err
		}
	}

	// update the "Date" header every time PrepareHeader is called
	if c.acctConfig.SendAsUTC {
		c.header.SetDate(time.Now().UTC())
	} else {
		c.header.SetDate(time.Now())
	}

	return c.header, nil
}

func getMessageIdHostname(c *Composer) (string, error) {
	if c.acctConfig.SendWithHostname {
		return os.Hostname()
	}
	addrs, err := c.header.AddressList("from")
	if err != nil {
		return "", err
	}
	_, domain, found := strings.Cut(addrs[0].Address, "@")
	if !found {
		return "", fmt.Errorf("Invalid address %q", addrs[0])
	}
	return domain, nil
}

func (c *Composer) parseEmbeddedHeader() (*mail.Header, error) {
	_, err := c.email.Seek(0, io.SeekStart)
	if err != nil {
		return nil, errors.Wrap(err, "Seek")
	}

	buf := bytes.NewBuffer([]byte{})
	_, err = io.Copy(buf, c.email)
	if err != nil {
		return nil, fmt.Errorf("mail.ReadMessageCopy: %w", err)
	}
	if config.Compose.LFEditor {
		bytes.ReplaceAll(buf.Bytes(), []byte{'\n'}, []byte{'\r', '\n'})
	}

	msg, err := mail.CreateReader(buf)
	if errors.Is(err, io.EOF) { // completely empty
		h := mail.HeaderFromMap(make(map[string][]string))
		return &h, nil
	} else if err != nil {
		return nil, fmt.Errorf("mail.ReadMessage: %w", err)
	}
	return &msg.Header, nil
}

func getRecipientsEmail(c *Composer) ([]string, error) {
	h, err := c.PrepareHeader()
	if err != nil {
		return nil, errors.Wrap(err, "PrepareHeader")
	}

	// collect all 'recipients' from header (to:, cc:, bcc:)
	rcpts := make(map[string]bool)
	for _, key := range []string{"to", "cc", "bcc"} {
		list, err := h.AddressList(key)
		if err != nil {
			continue
		}
		for _, entry := range list {
			if entry != nil {
				rcpts[entry.Address] = true
			}
		}
	}

	// return email addresses as string slice
	results := []string{}
	for email := range rcpts {
		results = append(results, email)
	}
	return results, nil
}

func (c *Composer) Signer() (string, error) {
	signer := ""

	if c.acctConfig.PgpKeyId != "" {
		// get key from explicitly set keyid
		signer = c.acctConfig.PgpKeyId
	} else {
		// get signer from `from` header
		from, err := c.header.AddressList("from")
		if err != nil {
			return "", err
		}

		if len(from) > 0 {
			signer = from[0].Address
		} else {
			// fall back to address from config
			signer = c.acctConfig.From.Address
		}
	}

	return signer, nil
}

func (c *Composer) WriteMessage(header *mail.Header, writer io.Writer) error {
	if c.sign || c.encrypt {

		var signedHeader mail.Header
		signedHeader.SetContentType("text/plain", nil)

		var buf bytes.Buffer
		var cleartext io.WriteCloser
		var err error

		signer := ""
		if c.sign {
			signer, err = c.Signer()
			if err != nil {
				return errors.Wrap(err, "Signer")
			}
		}

		if c.encrypt {
			rcpts, err := getRecipientsEmail(c)
			if err != nil {
				return err
			}
			cleartext, err = c.aerc.Crypto.Encrypt(&buf, rcpts, signer, c.aerc.DecryptKeys, header)
			if err != nil {
				return err
			}
		} else {
			cleartext, err = c.aerc.Crypto.Sign(&buf, signer, c.aerc.DecryptKeys, header)
			if err != nil {
				return err
			}
		}

		err = writeMsgImpl(c, &signedHeader, cleartext)
		if err != nil {
			return err
		}
		err = cleartext.Close()
		if err != nil {
			return err
		}
		_, err = io.Copy(writer, &buf)
		if err != nil {
			return fmt.Errorf("failed to write message: %w", err)
		}
		return nil

	} else {
		return writeMsgImpl(c, header, writer)
	}
}

func (c *Composer) ShouldWarnAttachment() bool {
	regex := config.Compose.NoAttachmentWarning

	if regex == nil || len(c.attachments) > 0 {
		return false
	}

	body, err := c.GetBody()
	if err != nil {
		log.Warnf("failed to check for a forgotten attachment: %v", err)
		return true
	}

	return regex.Match(body.Bytes())
}

func (c *Composer) ShouldWarnSubject() bool {
	if !config.Compose.EmptySubjectWarning {
		return false
	}

	// ignore errors because the raw header field is sufficient here
	subject, _ := c.header.Subject()
	return len(subject) == 0
}

func writeMsgImpl(c *Composer, header *mail.Header, writer io.Writer) error {
	mimeParams := map[string]string{"Charset": "UTF-8"}
	if config.Compose.FormatFlowed {
		mimeParams["Format"] = "Flowed"
	}
	body, err := c.GetBody()
	if err != nil {
		return err
	}
	if len(c.attachments) == 0 && len(c.textParts) == 0 {
		// no attachments
		return writeInlineBody(header, body, writer, mimeParams)
	} else {
		// with attachments
		w, err := mail.CreateWriter(writer, *header)
		if err != nil {
			return errors.Wrap(err, "CreateWriter")
		}
		newPart, err := lib.NewPart("text/plain", mimeParams, body)
		if err != nil {
			return err
		}
		parts := []*lib.Part{newPart}
		if err := writeMultipartBody(append(parts, c.textParts...), w); err != nil {
			return errors.Wrap(err, "writeMultipartBody")
		}
		for _, a := range c.attachments {
			if err := a.WriteTo(w); err != nil {
				return errors.Wrap(err, "writeAttachment")
			}
		}
		w.Close()
	}
	return nil
}

func writeInlineBody(
	header *mail.Header,
	body io.Reader,
	writer io.Writer,
	mimeParams map[string]string,
) error {
	header.SetContentType("text/plain", mimeParams)
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
func writeMultipartBody(parts []*lib.Part, w *mail.Writer) error {
	bi, err := w.CreateInline()
	if err != nil {
		return errors.Wrap(err, "CreateInline")
	}
	defer bi.Close()

	for _, part := range parts {
		bh := mail.InlineHeader{}
		bh.SetContentType(part.MimeType, part.Params)
		bw, err := bi.CreatePart(bh)
		if err != nil {
			return errors.Wrap(err, "CreatePart")
		}
		defer bw.Close()
		if _, err := io.Copy(bw, part.NewReader()); err != nil {
			return errors.Wrap(err, "io.Copy")
		}
	}

	return nil
}

func (c *Composer) GetAttachments() []string {
	var names []string
	for _, a := range c.attachments {
		names = append(names, a.Name())
	}
	return names
}

func (c *Composer) AddAttachment(path string) {
	c.attachments = append(c.attachments, lib.NewFileAttachment(path))
	c.resetReview()
}

func (c *Composer) AddPartAttachment(name string, mimetype string,
	params map[string]string, body io.Reader,
) error {
	p, err := lib.NewPart(mimetype, params, body)
	if err != nil {
		return err
	}
	c.attachments = append(c.attachments, lib.NewPartAttachment(
		p, name,
	))
	c.resetReview()
	return nil
}

func (c *Composer) DeleteAttachment(name string) error {
	for i, a := range c.attachments {
		if a.Name() == name {
			c.attachments = append(c.attachments[:i], c.attachments[i+1:]...)
			c.resetReview()
			return nil
		}
	}

	return errors.New("attachment does not exist")
}

func (c *Composer) resetReview() {
	if c.review != nil {
		c.grid.Load().(*ui.Grid).RemoveChild(c.review)
		c.review = newReviewMessage(c, nil)
		c.grid.Load().(*ui.Grid).AddChild(c.review).At(3, 0)
	}
}

func (c *Composer) termEvent(event tcell.Event) bool {
	if event, ok := event.(*tcell.EventMouse); ok {
		if event.Buttons() == tcell.Button1 {
			c.FocusTerminal()
			return true
		}
	}
	return false
}

func (c *Composer) reopenEmailFile() error {
	name := c.email.Name()
	f, err := os.OpenFile(name, os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	err = c.email.Close()
	c.email = f
	return err
}

func (c *Composer) termClosed(err error) {
	c.Lock()
	defer c.Unlock()
	if c.editor == nil {
		return
	}
	if e := c.reopenEmailFile(); e != nil {
		c.aerc.PushError("Failed to reopen email file: " + e.Error())
	}
	editor := c.editor
	defer editor.Destroy()
	c.editor = nil
	c.focusable = c.focusable[:len(c.focusable)-1]
	if c.focused >= len(c.focusable) {
		c.focused = len(c.focusable) - 1
	}

	if editor.cmd.ProcessState.ExitCode() > 0 {
		c.Close()
		c.aerc.RemoveTab(c, true)
		c.aerc.PushError("Editor exited with error. Compose aborted!")
		return
	}

	if c.editHeaders {
		// parse embedded header when editor is closed
		embedHeader, err := c.parseEmbeddedHeader()
		if err != nil {
			c.aerc.PushError(err.Error())
			err := c.showTerminal()
			if err != nil {
				c.Close()
				c.aerc.RemoveTab(c, true)
				c.aerc.PushError(err.Error())
			}
			return
		}
		for _, h := range c.headerOrder() {
			if embedHeader.Get(h) == "" {
				// user deleted header in text editor
				c.delEditor(h)
			}
		}
		hf := embedHeader.Fields()
		for hf.Next() {
			if hf.Value() != "" {
				c.addEditor(hf.Key(), hf.Value(), false)
			}
		}
	}

	// prepare review window
	c.review = newReviewMessage(c, err)
	c.updateGrid()
}

func (c *Composer) ShowTerminal(editHeaders bool) error {
	c.Lock()
	defer c.Unlock()
	if c.editor != nil {
		return nil
	}
	body, err := c.GetBody()
	if err != nil {
		return err
	}
	c.editHeaders = editHeaders
	err = c.setContents(body)
	if err != nil {
		return err
	}
	return c.showTerminal()
}

func (c *Composer) showTerminal() error {
	if c.editor != nil {
		c.editor.Destroy()
	}
	cmds := []string{
		config.Compose.Editor,
		os.Getenv("EDITOR"),
		"vi",
		"nano",
	}
	editorName, err := c.aerc.CmdFallbackSearch(cmds)
	if err != nil {
		c.acct.PushError(fmt.Errorf("could not start editor: %w", err))
	}
	editor := exec.Command("/bin/sh", "-c", editorName+" "+c.email.Name())
	c.editor, err = NewTerminal(editor)
	if err != nil {
		return err
	}
	c.editor.OnEvent = c.termEvent
	c.editor.OnClose = c.termClosed
	c.focusable = append(c.focusable, c.editor)
	c.review = nil
	c.updateGrid()
	if c.editHeaders {
		c.focusTerminalPriv()
	}
	return nil
}

func (c *Composer) PrevField() {
	c.Lock()
	defer c.Unlock()
	if c.editHeaders && c.editor != nil {
		return
	}
	c.focusActiveWidget(false)
	c.focused--
	if c.focused == -1 {
		c.focused = len(c.focusable) - 1
	}
	c.focusActiveWidget(true)
}

func (c *Composer) NextField() {
	c.Lock()
	defer c.Unlock()
	if c.editHeaders && c.editor != nil {
		return
	}
	c.focusActiveWidget(false)
	c.focused = (c.focused + 1) % len(c.focusable)
	c.focusActiveWidget(true)
}

func (c *Composer) FocusEditor(editor string) {
	c.Lock()
	defer c.Unlock()
	if c.editHeaders && c.editor != nil {
		return
	}
	c.focusEditor(editor)
}

func (c *Composer) focusEditor(editor string) {
	editor = strings.ToLower(editor)
	c.focusActiveWidget(false)
	for i, f := range c.focusable {
		e := f.(*headerEditor)
		if strings.ToLower(e.name) == editor {
			c.focused = i
			break
		}
	}
	c.focusActiveWidget(true)
}

// AddEditor appends a new header editor to the compose window.
func (c *Composer) AddEditor(header string, value string, appendHeader bool) error {
	c.Lock()
	defer c.Unlock()
	if c.editHeaders && c.editor != nil {
		return errors.New("header should be added directly in the text editor")
	}
	value = c.addEditor(header, value, appendHeader)
	if value == "" {
		c.focusEditor(header)
	}
	c.updateGrid()
	return nil
}

func (c *Composer) addEditor(header string, value string, appendHeader bool) string {
	var editor *headerEditor
	header = strings.ToLower(header)
	if e, ok := c.editors[header]; ok {
		e.storeValue() // flush modifications from the user to the header
		editor = e
	} else {
		uiConfig := c.acct.UiConfig()
		e := newHeaderEditor(header, c.header, uiConfig)
		if uiConfig.CompletionPopovers {
			e.input.TabComplete(
				c.completer.ForHeader(header),
				uiConfig.CompletionDelay,
				uiConfig.CompletionMinChars,
			)
		}
		c.editors[header] = e
		c.layout = append(c.layout, []string{header})
		switch {
		case len(c.focusable) == 0:
			c.focusable = []ui.MouseableDrawableInteractive{e}
		case c.editor != nil:
			// Insert focus of new editor before terminal editor
			c.focusable = append(
				c.focusable[:len(c.focusable)-1],
				e,
				c.focusable[len(c.focusable)-1],
			)
		default:
			c.focusable = append(
				c.focusable[:len(c.focusable)-1],
				e,
			)
		}
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
	return value
}

// DelEditor removes a header editor from the compose window.
func (c *Composer) DelEditor(header string) error {
	c.Lock()
	defer c.Unlock()
	if c.editHeaders && c.editor != nil {
		return errors.New("header should be removed directly in the text editor")
	}
	c.delEditor(header)
	c.updateGrid()
	return nil
}

func (c *Composer) delEditor(header string) {
	header = strings.ToLower(header)
	c.header.Del(header)
	editor, ok := c.editors[header]
	if !ok {
		return
	}

	var layout HeaderLayout = make([][]string, 0, len(c.layout))
	for _, row := range c.layout {
		r := make([]string, 0, len(row))
		for _, h := range row {
			if h != header {
				r = append(r, h)
			}
		}
		if len(r) > 0 {
			layout = append(layout, r)
		}
	}
	c.layout = layout

	focusable := make([]ui.MouseableDrawableInteractive, 0, len(c.focusable)-1)
	for i, f := range c.focusable {
		if f == editor {
			if c.focused > 0 && c.focused >= i {
				c.focused--
			}
		} else {
			focusable = append(focusable, f)
		}
	}
	c.focusable = focusable
	c.focusActiveWidget(true)

	delete(c.editors, header)
}

// updateGrid should be called when the underlying header layout is changed.
func (c *Composer) updateGrid() {
	grid := ui.NewGrid().Columns([]ui.GridSpec{
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	})

	if c.editHeaders && c.review == nil {
		grid.Rows([]ui.GridSpec{
			// 0: editor
			{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
		})
		if c.editor != nil {
			grid.AddChild(c.editor).At(0, 0)
		}
		c.grid.Store(grid)
		return
	}

	heditors, height := c.layout.grid(
		func(h string) ui.Drawable {
			return c.editors[h]
		},
	)

	crHeight := 0
	if c.sign || c.encrypt {
		crHeight = 1
	}
	grid.Rows([]ui.GridSpec{
		// 0: headers
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(height)},
		// 1: crypto status
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(crHeight)},
		// 2: filler line
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)},
		// 3: editor or review
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	})

	borderStyle := c.acct.UiConfig().GetStyle(config.STYLE_BORDER)
	borderChar := c.acct.UiConfig().BorderCharHorizontal
	grid.AddChild(heditors).At(0, 0)
	grid.AddChild(c.crypto).At(1, 0)
	grid.AddChild(ui.NewFill(borderChar, borderStyle)).At(2, 0)
	if c.review != nil {
		grid.AddChild(c.review).At(3, 0)
	} else if c.editor != nil {
		grid.AddChild(c.editor).At(3, 0)
	}
	c.heditors.Store(heditors)
	c.grid.Store(grid)
}

type headerEditor struct {
	name     string
	header   *mail.Header
	focused  bool
	input    *ui.TextInput
	uiConfig *config.UIConfig
}

func newHeaderEditor(name string, h *mail.Header,
	uiConfig *config.UIConfig,
) *headerEditor {
	he := &headerEditor{
		input:    ui.NewTextInput("", uiConfig),
		name:     name,
		header:   h,
		uiConfig: uiConfig,
	}
	he.loadValue()
	return he
}

// extractHumanHeaderValue extracts the human readable string for key from the
// header. If a parsing error occurs the raw value is returned
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

// loadValue loads the value of he.name form the underlying header
// the value is decoded and meant for human consumption.
// decoding issues are ignored and return their raw values
func (he *headerEditor) loadValue() {
	he.input.Set(extractHumanHeaderValue(he.name, he.header))
	ui.Invalidate()
}

// storeValue writes the current state back to the underlying header.
// errors are ignored
func (he *headerEditor) storeValue() {
	val := he.input.String()
	switch strings.ToLower(he.name) {
	case "to", "from", "cc", "bcc":
		if strings.TrimSpace(val) == "" {
			// if header is empty, delete it
			he.header.Del(he.name)
			return
		}
		list, err := mail.ParseAddressList(val)
		if err == nil {
			he.header.SetAddressList(he.name, list)
		} else {
			// garbage, but it'll blow up upon sending and the user can
			// fix the issue
			he.header.SetText(he.name, val)
		}
	default:
		he.header.SetText(he.name, val)
	}
	if strings.ToLower(he.name) == "from" {
		he.header.Del("message-id")
	}
}

func (he *headerEditor) Draw(ctx *ui.Context) {
	name := textproto.CanonicalMIMEHeaderKey(he.name)
	// Extra character to put a blank cell between the header and the input
	size := runewidth.StringWidth(name+":") + 1
	defaultStyle := he.uiConfig.GetStyle(config.STYLE_DEFAULT)
	headerStyle := he.uiConfig.GetStyle(config.STYLE_HEADER)
	ctx.Fill(0, 0, size, ctx.Height(), ' ', defaultStyle)
	ctx.Printf(0, 0, headerStyle, "%s:", name)
	he.input.Draw(ctx.Subcontext(size, 0, ctx.Width()-size, 1))
}

func (he *headerEditor) MouseEvent(localX int, localY int, event tcell.Event) {
	if event, ok := event.(*tcell.EventMouse); ok {
		if event.Buttons() == tcell.Button1 {
			he.focused = true
		}

		width := runewidth.StringWidth(he.name + " ")
		if localX >= width {
			he.input.MouseEvent(localX-width, localY, event)
		}
	}
}

func (he *headerEditor) Invalidate() {
	ui.Invalidate()
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

func (he *headerEditor) OnFocusLost(fn func()) {
	he.input.OnFocusLost(func(_ *ui.TextInput) {
		fn()
	})
}

type reviewMessage struct {
	composer *Composer
	grid     *ui.Grid
}

func newReviewMessage(composer *Composer, err error) *reviewMessage {
	bindings := config.Binds.ComposeReview.ForAccount(
		composer.acctConfig.Name,
	)

	reviewCommands := [][]string{
		{":send<enter>", "Send", ""},
		{":edit<enter>", "Edit", ""},
		{":attach<space>", "Add attachment", ""},
		{":detach<space>", "Remove attachment", ""},
		{":postpone<enter>", "Postpone", ""},
		{":preview<enter>", "Preview message", ""},
		{":abort<enter>", "Abort (discard message, no confirmation)", ""},
		{":choose -o d discard abort -o p postpone postpone<enter>", "Abort or postpone", ""},
	}
	knownCommands := len(reviewCommands)
	var actions []string
	for _, binding := range bindings.Bindings {
		inputs := config.FormatKeyStrokes(binding.Input)
		outputs := config.FormatKeyStrokes(binding.Output)
		found := false
		for i, rcmd := range reviewCommands {
			if outputs == rcmd[0] {
				found = true
				if reviewCommands[i][2] == "" {
					reviewCommands[i][2] = inputs
				} else {
					reviewCommands[i][2] += ", " + inputs
				}
				break
			}
		}
		if !found {
			rcmd := []string{outputs, "", inputs}
			reviewCommands = append(reviewCommands, rcmd)
		}
	}
	unknownCommands := reviewCommands[knownCommands:]
	sort.Slice(unknownCommands, func(i, j int) bool {
		return unknownCommands[i][2] < unknownCommands[j][2]
	})

	longest := 0
	for _, rcmd := range reviewCommands {
		if len(rcmd[2]) > longest {
			longest = len(rcmd[2])
		}
	}

	width := longest
	if longest < 6 {
		width = 6
	}
	widthstr := strconv.Itoa(width)

	for _, rcmd := range reviewCommands {
		if rcmd[2] != "" {
			actions = append(actions, fmt.Sprintf("  %-"+widthstr+"s  %-40s  %s",
				rcmd[2], rcmd[1], rcmd[0]))
		}
	}

	spec := []ui.GridSpec{
		{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)},
	}
	for i := 0; i < len(actions)-1; i++ {
		spec = append(spec, ui.GridSpec{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)})
	}
	spec = append(spec, ui.GridSpec{Strategy: ui.SIZE_EXACT, Size: ui.Const(2)})
	spec = append(spec, ui.GridSpec{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)})
	for i := 0; i < len(composer.attachments)-1; i++ {
		spec = append(spec, ui.GridSpec{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)})
	}
	if len(composer.textParts) > 0 {
		spec = append(spec, ui.GridSpec{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)})
		spec = append(spec, ui.GridSpec{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)})
		for i := 0; i < len(composer.textParts); i++ {
			spec = append(spec, ui.GridSpec{Strategy: ui.SIZE_EXACT, Size: ui.Const(1)})
		}
	}
	// make the last element fill remaining space
	spec = append(spec, ui.GridSpec{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)})

	grid := ui.NewGrid().Rows(spec).Columns([]ui.GridSpec{
		{Strategy: ui.SIZE_WEIGHT, Size: ui.Const(1)},
	})

	uiConfig := composer.acct.UiConfig()

	if err != nil {
		grid.AddChild(ui.NewText(err.Error(), uiConfig.GetStyle(config.STYLE_ERROR)))
		grid.AddChild(ui.NewText("Press [q] to close this tab.",
			uiConfig.GetStyle(config.STYLE_DEFAULT))).At(1, 0)
	} else {
		grid.AddChild(ui.NewText("Send this email?",
			uiConfig.GetStyle(config.STYLE_TITLE))).At(0, 0)
		i := 1
		for _, action := range actions {
			grid.AddChild(ui.NewText(action,
				uiConfig.GetStyle(config.STYLE_DEFAULT))).At(i, 0)
			i += 1
		}
		grid.AddChild(ui.NewText("Attachments:",
			uiConfig.GetStyle(config.STYLE_TITLE))).At(i, 0)
		i += 1
		if len(composer.attachments) == 0 {
			grid.AddChild(ui.NewText("(none)",
				uiConfig.GetStyle(config.STYLE_DEFAULT))).At(i, 0)
			i += 1
		} else {
			for _, a := range composer.attachments {
				grid.AddChild(ui.NewText(a.Name(), uiConfig.GetStyle(config.STYLE_DEFAULT))).
					At(i, 0)
				i += 1
			}
		}
		if len(composer.textParts) > 0 {
			grid.AddChild(ui.NewText("Parts:",
				uiConfig.GetStyle(config.STYLE_TITLE))).At(i, 0)
			i += 1
			grid.AddChild(ui.NewText("text/plain", uiConfig.GetStyle(config.STYLE_DEFAULT))).At(i, 0)
			i += 1
			for _, p := range composer.textParts {
				err := composer.updateMultipart(p)
				if err != nil {
					msg := fmt.Sprintf("%s error: %s", p.MimeType, err)
					grid.AddChild(ui.NewText(msg,
						uiConfig.GetStyle(config.STYLE_ERROR))).At(i, 0)
				} else {
					grid.AddChild(ui.NewText(p.MimeType,
						uiConfig.GetStyle(config.STYLE_DEFAULT))).At(i, 0)
				}
				i += 1
			}

		}
	}

	return &reviewMessage{
		composer: composer,
		grid:     grid,
	}
}

func (c *Composer) updateMultipart(p *lib.Part) error {
	command, found := config.Converters[p.MimeType]
	if !found {
		// unreachable
		return fmt.Errorf("no command defined for mime/type")
	}
	// reset part body to avoid it leaving outdated if the command fails
	p.Data = nil
	body, err := c.GetBody()
	if err != nil {
		return errors.Wrap(err, "GetBody")
	}
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdin = body
	out, err := cmd.Output()
	if err != nil {
		var stderr string
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			// append the first 30 chars of stderr if any
			stderr = strings.Trim(string(ee.Stderr), " \t\n\r")
			stderr = strings.ReplaceAll(stderr, "\n", "; ")
			if stderr != "" {
				stderr = fmt.Sprintf(": %.30s", stderr)
			}
		}
		return fmt.Errorf("%s: %w%s", command, err, stderr)
	}
	p.Data = out
	return nil
}

func (rm *reviewMessage) Invalidate() {
	ui.Invalidate()
}

func (rm *reviewMessage) Draw(ctx *ui.Context) {
	rm.grid.Draw(ctx)
}

type cryptoStatus struct {
	title         string
	status        *ui.Text
	uiConfig      *config.UIConfig
	signKey       string
	setEncOneShot bool
}

func newCryptoStatus(uiConfig *config.UIConfig) *cryptoStatus {
	defaultStyle := uiConfig.GetStyle(config.STYLE_DEFAULT)
	return &cryptoStatus{
		title:         "Security",
		status:        ui.NewText("", defaultStyle),
		uiConfig:      uiConfig,
		signKey:       "",
		setEncOneShot: true,
	}
}

func (cs *cryptoStatus) Draw(ctx *ui.Context) {
	// Extra character to put a blank cell between the header and the input
	size := runewidth.StringWidth(cs.title+":") + 1
	defaultStyle := cs.uiConfig.GetStyle(config.STYLE_DEFAULT)
	titleStyle := cs.uiConfig.GetStyle(config.STYLE_HEADER)
	ctx.Fill(0, 0, size, ctx.Height(), ' ', defaultStyle)
	ctx.Printf(0, 0, titleStyle, "%s:", cs.title)
	cs.status.Draw(ctx.Subcontext(size, 0, ctx.Width()-size, 1))
}

func (cs *cryptoStatus) Invalidate() {
	ui.Invalidate()
}

func (c *Composer) checkEncryptionKeys(_ string) bool {
	rcpts, err := getRecipientsEmail(c)
	if err != nil {
		// checkEncryptionKeys gets registered as a callback and must
		// explicitly call c.SetEncrypt(false) when encryption is not possible
		c.SetEncrypt(false)
		st := fmt.Sprintf("Cannot encrypt: %v", err)
		c.aerc.statusline.PushError(st)
		return false
	}
	var mk []string
	for _, rcpt := range rcpts {
		key, err := c.aerc.Crypto.GetKeyId(rcpt)
		if err != nil || key == "" {
			mk = append(mk, rcpt)
		}
	}

	encrypt := true
	switch {
	case len(mk) > 0:
		c.SetEncrypt(false)
		st := fmt.Sprintf("Cannot encrypt, missing keys: %s", strings.Join(mk, ", "))
		if c.Config().PgpOpportunisticEncrypt {
			switch c.Config().PgpErrorLevel {
			case config.PgpErrorLevelWarn:
				c.aerc.statusline.PushWarning(st)
				return false
			case config.PgpErrorLevelNone:
				return false
			case config.PgpErrorLevelError:
				// Continue to the default
			}
		}
		c.aerc.statusline.PushError(st)
		encrypt = false
	case len(rcpts) == 0:
		encrypt = false
	}

	// If callbacks were registered, encrypt will be set when user removes
	// recipients with missing keys
	c.encrypt = encrypt
	err = c.updateCrypto()
	if err != nil {
		log.Warnf("failed update crypto: %v", err)
	}
	return true
}

// setTitle executes the title template and sets the tab title
func (c *Composer) setTitle() {
	if c.Tab == nil {
		return
	}

	header := c.header.Copy()
	// Get subject direct from the textinput
	subject, ok := c.editors["subject"]
	if ok {
		header.SetSubject(subject.input.String())
	}
	if header.Get("subject") == "" {
		header.SetSubject("New Email")
	}

	data := state.NewDataSetter()
	data.SetAccount(c.acctConfig)
	data.SetFolder(c.acct.Directories().SelectedDirectory())
	data.SetHeaders(&header, c.parent)

	var buf bytes.Buffer
	err := templates.Render(c.acct.UiConfig().TabTitleComposer, &buf,
		data.Data())
	if err != nil {
		c.acct.PushError(err)
		return
	}
	c.Tab.SetTitle(buf.String())
}
