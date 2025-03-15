package state

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/lib/xdg"
	"git.sr.ht/~rjarry/aerc/models"
	"github.com/danwakefield/fnmatch"
	sortthread "github.com/emersion/go-imap-sortthread"
	"github.com/emersion/go-message/mail"
)

type Composer interface {
	AddAttachment(string)
}

type DataSetter interface {
	Data() models.TemplateData
	SetHeaders(*mail.Header, *models.OriginalMail)
	SetInfo(*models.MessageInfo, int, bool)
	SetVisual(bool)
	SetThreading(ThreadInfo)
	SetComposer(Composer)
	SetAccount(*config.AccountConfig)
	SetFolder(*models.Directory)
	SetRUE([]string, func(string) (int, int, int))
	SetState(s *AccountState)
	SetPendingKeys([]config.KeyStroke)
}

type ThreadInfo struct {
	SameSubject bool
	Prefix      string
	Count       int
	Unread      int
	Folded      bool
	Context     bool
	Orphan      bool
}

type templateData struct {
	// only available when composing/replying/forwarding
	headers *mail.Header
	// only available when replying with a quote
	parent *models.OriginalMail
	// only available for the message list
	info   *models.MessageInfo
	marked bool
	msgNum int
	visual bool

	// message list threading
	threadInfo ThreadInfo

	// selected account
	account     *config.AccountConfig
	myAddresses map[string]bool
	folder      *models.Directory // selected folder
	folders     []string
	getRUEcount func(string) (int, int, int)

	state       *AccountState
	pendingKeys []config.KeyStroke

	composer Composer
}

func NewDataSetter() DataSetter {
	return &templateData{}
}

// Data returns the template data
func (d *templateData) Data() models.TemplateData {
	return d
}

// only used for compose/reply/forward
func (d *templateData) SetHeaders(h *mail.Header, o *models.OriginalMail) {
	d.headers = h
	d.parent = o
}

// only used for message list templates
func (d *templateData) SetInfo(info *models.MessageInfo, num int, marked bool,
) {
	d.info = info
	d.msgNum = num
	d.marked = marked
}

func (d *templateData) SetVisual(visual bool) {
	d.visual = visual
}

func (d *templateData) SetThreading(info ThreadInfo) {
	d.threadInfo = info
}

func (d *templateData) SetAccount(acct *config.AccountConfig) {
	d.account = acct
	d.myAddresses = make(map[string]bool)
	if acct != nil {
		d.myAddresses[acct.From.Address] = true
		for _, addr := range acct.Aliases {
			d.myAddresses[addr.Address] = true
		}
	}
}

func (d *templateData) SetFolder(folder *models.Directory) {
	d.folder = folder
}

func (d *templateData) SetComposer(c Composer) {
	d.composer = c
}

func (d *templateData) SetRUE(folders []string,
	cb func(string) (int, int, int),
) {
	d.folders = folders
	d.getRUEcount = cb
}

func (d *templateData) SetState(state *AccountState) {
	d.state = state
}

func (d *templateData) SetPendingKeys(keys []config.KeyStroke) {
	d.pendingKeys = keys
}

func (d *templateData) Attach(s string) string {
	if d.composer != nil {
		d.composer.AddAttachment(s)
		return ""
	}
	return fmt.Sprintf("Failed to attach: %s", s)
}

func (d *templateData) Account() string {
	if d.account != nil {
		return d.account.Name
	}
	return ""
}

func (d *templateData) AccountBackend() string {
	if d.account != nil {
		return d.account.Backend
	}
	return ""
}

func (d *templateData) AccountFrom() *mail.Address {
	if d.account != nil {
		return d.account.From
	}
	return nil
}

func (d *templateData) Folder() string {
	if d.folder != nil {
		return d.folder.Name
	}
	return ""
}

func (d *templateData) Role() string {
	if d.folder != nil {
		return string(d.folder.Role)
	}
	return ""
}

func (d *templateData) ui() *config.UIConfig {
	return config.Ui.ForAccount(d.Account()).ForFolder(d.Folder())
}

func (d *templateData) To() []*mail.Address {
	var to []*mail.Address
	switch {
	case d.info != nil && d.info.Envelope != nil:
		to = d.info.Envelope.To
	case d.headers != nil:
		to, _ = d.headers.AddressList("to")
	}
	return to
}

func (d *templateData) Cc() []*mail.Address {
	var cc []*mail.Address
	switch {
	case d.info != nil && d.info.Envelope != nil:
		cc = d.info.Envelope.Cc
	case d.headers != nil:
		cc, _ = d.headers.AddressList("cc")
	}
	return cc
}

func (d *templateData) Bcc() []*mail.Address {
	var bcc []*mail.Address
	switch {
	case d.info != nil && d.info.Envelope != nil:
		bcc = d.info.Envelope.Bcc
	case d.headers != nil:
		bcc, _ = d.headers.AddressList("bcc")
	}
	return bcc
}

func (d *templateData) From() []*mail.Address {
	var from []*mail.Address
	switch {
	case d.info != nil && d.info.Envelope != nil:
		from = d.info.Envelope.From
	case d.headers != nil:
		from, _ = d.headers.AddressList("from")
	}
	return from
}

func (d *templateData) Peer() []*mail.Address {
	var from, to []*mail.Address
	switch {
	case d.info != nil && d.info.Envelope != nil:
		from = d.info.Envelope.From
		to = d.info.Envelope.To
	case d.headers != nil:
		from, _ = d.headers.AddressList("from")
		to, _ = d.headers.AddressList("to")
	}
	for _, addr := range from {
		for myAddr := range d.myAddresses {
			if fnmatch.Match(myAddr, addr.Address, 0) {
				return to
			}
		}
	}
	return from
}

func (d *templateData) ReplyTo() []*mail.Address {
	var replyTo []*mail.Address
	switch {
	case d.info != nil && d.info.Envelope != nil:
		replyTo = d.info.Envelope.ReplyTo
	case d.headers != nil:
		replyTo, _ = d.headers.AddressList("reply-to")
	}
	return replyTo
}

func (d *templateData) Date() time.Time {
	var date time.Time
	switch {
	case d.info != nil && d.info.Envelope != nil:
		date = d.info.Envelope.Date
	case d.info != nil:
		date = d.info.InternalDate
	default:
		date = time.Now()
	}
	return date
}

func (d *templateData) DateAutoFormat(date time.Time) string {
	if date.IsZero() {
		return ""
	}
	ui := d.ui()
	year := date.Year()
	day := date.YearDay()
	now := time.Now()
	thisYear := now.Year()
	thisDay := now.YearDay()
	fmt := ui.TimestampFormat
	if year == thisYear {
		switch {
		case day == thisDay && ui.ThisDayTimeFormat != "":
			fmt = ui.ThisDayTimeFormat
		case day > thisDay-7 && ui.ThisWeekTimeFormat != "":
			fmt = ui.ThisWeekTimeFormat
		case ui.ThisYearTimeFormat != "":
			fmt = ui.ThisYearTimeFormat
		}
	}
	return date.Format(fmt)
}

func (d *templateData) Header(name string) string {
	var h *mail.Header
	switch {
	case d.headers != nil:
		h = d.headers
	case d.info != nil && d.info.RFC822Headers != nil:
		h = d.info.RFC822Headers
	default:
		return ""
	}
	text, err := h.Text(name)
	if err != nil {
		text = h.Get(name)
	}
	return text
}

func (d *templateData) ThreadPrefix() string {
	return d.threadInfo.Prefix
}

func (d *templateData) ThreadCount() int {
	return d.threadInfo.Count
}

func (d *templateData) ThreadUnread() int {
	return d.threadInfo.Unread
}

func (d *templateData) ThreadFolded() bool {
	return d.threadInfo.Folded
}

func (d *templateData) ThreadContext() bool {
	return d.threadInfo.Context
}

func (d *templateData) ThreadOrphan() bool {
	return d.threadInfo.Orphan
}

func (d *templateData) Subject() string {
	var subject string
	switch {
	case d.info != nil && d.info.Envelope != nil:
		subject = d.info.Envelope.Subject
	case d.headers != nil:
		subject = d.Header("subject")
	}
	if d.threadInfo.SameSubject {
		subject = ""
	} else if subject == "" {
		subject = config.Ui.EmptySubject
	}
	return subject
}

func (d *templateData) SubjectBase() string {
	var subject string
	switch {
	case d.info != nil && d.info.Envelope != nil:
		subject = d.info.Envelope.Subject
	case d.headers != nil:
		subject = d.Header("subject")
	}
	base, _ := sortthread.GetBaseSubject(subject)
	return base
}

func (d *templateData) Number() int {
	return d.msgNum
}

func (d *templateData) Labels() []string {
	if d.info == nil {
		return nil
	}
	return d.info.Labels
}

func (d *templateData) Filename() string {
	if d.info == nil {
		return ""
	}
	if (d.info.Filenames != nil) && len(d.info.Filenames) > 0 {
		return d.info.Filenames[0]
	}
	return ""
}

func (d *templateData) Filenames() []string {
	if d.info == nil {
		return nil
	}
	return d.info.Filenames
}

func (d *templateData) Flags() []string {
	var flags []string
	if d.info == nil {
		return flags
	}

	switch {
	case d.info.Flags.Has(models.SeenFlag | models.AnsweredFlag):
		flags = append(flags, d.ui().IconReplied) // message has been replied to
	case d.info.Flags.Has(models.SeenFlag):
		break
	case d.info.Flags.Has(models.RecentFlag):
		flags = append(flags, d.ui().IconNew) // message is unread and new
	default:
		flags = append(flags, d.ui().IconOld) // message is unread and old
	}
	if d.info.Flags.Has(models.DraftFlag) {
		flags = append(flags, d.ui().IconDraft)
	}
	if d.info.Flags.Has(models.DeletedFlag) {
		flags = append(flags, d.ui().IconDeleted)
	}
	if d.info.Flags.Has(models.ForwardedFlag) {
		flags = append(flags, d.ui().IconForwarded)
	}
	if d.info.BodyStructure != nil {
		for _, bS := range d.info.BodyStructure.Parts {
			if strings.ToLower(bS.Disposition) == "attachment" {
				flags = append(flags, d.ui().IconAttachment)
				break
			}
		}
	}
	if d.info.Flags.Has(models.FlaggedFlag) {
		flags = append(flags, d.ui().IconFlagged)
	}
	if d.marked {
		flags = append(flags, d.ui().IconMarked)
	}
	return flags
}

func (d *templateData) IsReplied() bool {
	if d.info != nil && d.info.Flags.Has(models.AnsweredFlag) {
		return true
	}
	return false
}

func (d *templateData) IsForwarded() bool {
	if d.info != nil && d.info.Flags.Has(models.ForwardedFlag) {
		return true
	}
	return false
}

func (d *templateData) HasAttachment() bool {
	if d.info != nil && d.info.BodyStructure != nil {
		for _, bS := range d.info.BodyStructure.Parts {
			if strings.ToLower(bS.Disposition) == "attachment" {
				return true
			}
		}
	}
	return false
}

func (d *templateData) IsRecent() bool {
	if d.info != nil && d.info.Flags.Has(models.RecentFlag) {
		return true
	}
	return false
}

func (d *templateData) IsUnread() bool {
	if d.info != nil && !d.info.Flags.Has(models.SeenFlag) {
		return true
	}
	return false
}

func (d *templateData) IsFlagged() bool {
	if d.info != nil && d.info.Flags.Has(models.FlaggedFlag) {
		return true
	}
	return false
}

func (d *templateData) IsDraft() bool {
	if d.info != nil && d.info.Flags.Has(models.DraftFlag) {
		return true
	}
	return false
}

func (d *templateData) IsMarked() bool {
	return d.marked
}

func (d *templateData) MessageId() string {
	if d.info == nil || d.info.Envelope == nil {
		return ""
	}
	return d.info.Envelope.MessageId
}

func (d *templateData) Size() int {
	if d.info == nil || d.info.Envelope == nil {
		return 0
	}
	return int(d.info.Size)
}

func (d *templateData) OriginalText() string {
	if d.parent == nil {
		return ""
	}
	return d.parent.Text
}

func (d *templateData) OriginalDate() time.Time {
	if d.parent == nil {
		return time.Time{}
	}
	return d.parent.Date
}

func (d *templateData) OriginalFrom() []*mail.Address {
	if d.parent == nil || d.parent.RFC822Headers == nil {
		return nil
	}
	from, _ := d.parent.RFC822Headers.AddressList("from")
	return from
}

func (d *templateData) OriginalMIMEType() string {
	if d.parent == nil {
		return ""
	}
	return d.parent.MIMEType
}

func (d *templateData) OriginalHeader(name string) string {
	if d.parent == nil || d.parent.RFC822Headers == nil {
		return ""
	}
	text, err := d.parent.RFC822Headers.Text(name)
	if err != nil {
		text = d.parent.RFC822Headers.Get(name)
	}
	return text
}

func (d *templateData) rue(folders ...string) (int, int, int) {
	var recent, unread, exists int
	if d.getRUEcount != nil {
		if len(folders) == 0 {
			folders = d.folders
		}
		for _, dir := range folders {
			r, u, e := d.getRUEcount(dir)
			recent += r
			unread += u
			exists += e
		}
	}
	return recent, unread, exists
}

func (d *templateData) Recent(folders ...string) int {
	r, _, _ := d.rue(folders...)
	return r
}

func (d *templateData) Unread(folders ...string) int {
	_, u, _ := d.rue(folders...)
	return u
}

func (d *templateData) Exists(folders ...string) int {
	_, _, e := d.rue(folders...)
	return e
}

func (d *templateData) RUE(folders ...string) string {
	r, u, e := d.rue(folders...)
	switch {
	case r > 0:
		return fmt.Sprintf("%d/%d/%d", r, u, e)
	case u > 0:
		return fmt.Sprintf("%d/%d", u, e)
	case e > 0:
		return fmt.Sprintf("%d", e)
	}
	return ""
}

func (d *templateData) Connected() bool {
	if d.state != nil {
		return d.state.Connected
	}
	return false
}

func (d *templateData) ConnectionInfo() string {
	switch {
	case d.state == nil:
		return ""
	case d.state.connActivity != "":
		return d.state.connActivity
	case d.state.Connected:
		return texter().Connected()
	default:
		return texter().Disconnected()
	}
}

func (d *templateData) ContentInfo() string {
	if d.state == nil {
		return ""
	}
	var content []string
	fldr := d.state.folderState(d.Folder())
	if fldr.FilterActivity != "" {
		content = append(content, fldr.FilterActivity)
	} else if fldr.Filter != "" {
		content = append(content, texter().FormatFilter(fldr.Filter))
	}
	if fldr.Search != "" {
		content = append(content, texter().FormatSearch(fldr.Search))
	}
	return strings.Join(content, config.Statusline.Separator)
}

func (d *templateData) StatusInfo() string {
	stat := d.ConnectionInfo()
	if content := d.ContentInfo(); content != "" {
		stat += config.Statusline.Separator + content
	}
	return stat
}

func (d *templateData) TrayInfo() string {
	if d.state == nil {
		return ""
	}
	var tray []string
	fldr := d.state.folderState(d.Folder())
	if fldr.Sorting {
		tray = append(tray, texter().Sorting())
	}
	if fldr.Threading {
		tray = append(tray, texter().Threading())
	}
	if d.state.passthrough {
		tray = append(tray, texter().Passthrough())
	}
	if d.visual {
		tray = append(tray, texter().Visual())
	}
	return strings.Join(tray, config.Statusline.Separator)
}

func (d *templateData) PendingKeys() string {
	return config.FormatKeyStrokes(d.pendingKeys)
}

func (d *templateData) Style(content, name string) string {
	cfg := config.Ui.ForAccount(d.Account())
	style := cfg.GetUserStyle(name)
	return ui.ApplyStyle(style, content)
}

func (d *templateData) StyleSwitch(content string, cases ...models.Case) string {
	for _, c := range cases {
		if c.Matches(content) {
			cfg := config.Ui.ForAccount(d.Account())
			style := cfg.GetUserStyle(c.Value())
			return ui.ApplyStyle(style, content)
		}
	}
	return content
}

func (d *templateData) StyleMap(elems []string, cases ...models.Case) []string {
	mapped := make([]string, 0, len(elems))
top:
	for _, e := range elems {
		for _, c := range cases {
			if c.Matches(e) {
				if c.Skip() {
					continue top
				}
				cfg := config.Ui.ForAccount(d.Account())
				style := cfg.GetUserStyle(c.Value())
				e = ui.ApplyStyle(style, e)
				break
			}
		}
		mapped = append(mapped, e)
	}
	return mapped
}

func (d *templateData) Signature() string {
	if d.account == nil {
		return ""
	}
	var signature []byte
	if d.account.SignatureCmd != "" {
		var err error
		signature, err = d.readSignatureFromCmd()
		if err != nil {
			var execErr *exec.ExitError
			if errors.As(err, &execErr) {
				log.Warnf("signature command failed with error (%d): %s", execErr.ExitCode(), execErr.Stderr)
			}
			signature = d.readSignatureFromFile()
		}
	} else {
		signature = d.readSignatureFromFile()
	}
	if len(bytes.TrimSpace(signature)) == 0 {
		return ""
	}
	signature = d.ensureSignatureDelimiter(signature)
	return string(signature)
}

func (d *templateData) readSignatureFromCmd() ([]byte, error) {
	sigCmd := d.account.SignatureCmd
	cmd := exec.Command("sh", "-c", sigCmd)
	env := os.Environ()
	if d.account != nil {
		env = append(env, fmt.Sprintf("AERC_ACCOUNT=%s", d.account.Name))
	}
	if d.folder != nil {
		env = append(env, fmt.Sprintf("AERC_FOLDER=%s", d.folder.Name))
	}
	cmd.Env = env
	signature, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return signature, nil
}

func (d *templateData) readSignatureFromFile() []byte {
	sigFile := d.account.SignatureFile
	if sigFile == "" {
		return nil
	}
	sigFile = xdg.ExpandHome(sigFile)
	signature, err := os.ReadFile(sigFile)
	if err != nil {
		log.Errorf(" Error loading signature from file: %v", sigFile)
		return nil
	}
	return signature
}

func (d *templateData) ensureSignatureDelimiter(signature []byte) []byte {
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
