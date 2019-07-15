package maildir

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"mime/quotedprintable"
	"strings"

	"github.com/emersion/go-maildir"
	"github.com/emersion/go-message"
	_ "github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"

	"git.sr.ht/~sircmpwn/aerc/models"
)

// A Message is an individual email inside of a maildir.Dir.
type Message struct {
	dir maildir.Dir
	uid uint32
	key string
}

// NewReader reads a message into memory and returns an io.Reader for it.
func (m Message) NewReader() (io.Reader, error) {
	f, err := m.dir.Open(m.key)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(b), nil
}

// Flags fetches the set of flags currently applied to the message.
func (m Message) Flags() ([]maildir.Flag, error) {
	return m.dir.Flags(m.key)
}

// SetFlags replaces the message's flags with a new set.
func (m Message) SetFlags(flags []maildir.Flag) error {
	return m.dir.SetFlags(m.key, flags)
}

// MarkRead either adds or removes the maildir.FlagSeen flag from the message.
func (m Message) MarkRead(seen bool) error {
	flags, err := m.Flags()
	if err != nil {
		return fmt.Errorf("could not read previous flags: %v", err)
	}
	if seen {
		flags = append(flags, maildir.FlagSeen)
		return m.SetFlags(flags)
	}
	var newFlags []maildir.Flag
	for _, flag := range flags {
		if flag != maildir.FlagSeen {
			newFlags = append(newFlags, flag)
		}
	}
	return m.SetFlags(newFlags)
}

// Remove deletes the email immediately.
func (m Message) Remove() error {
	return m.dir.Remove(m.key)
}

// MessageInfo populates a models.MessageInfo struct for the message.
func (m Message) MessageInfo() (*models.MessageInfo, error) {
	f, err := m.dir.Open(m.key)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	msg, err := message.Read(f)
	if err != nil {
		return nil, fmt.Errorf("could not read message: %v", err)
	}
	bs, err := parseEntityStructure(msg)
	if err != nil {
		return nil, fmt.Errorf("could not get structure: %v", err)
	}
	env, err := parseEnvelope(&mail.Header{msg.Header})
	if err != nil {
		return nil, fmt.Errorf("could not get envelope: %v", err)
	}
	flags, err := m.Flags()
	if err != nil {
		return nil, fmt.Errorf("could not read flags: %v", err)
	}
	return &models.MessageInfo{
		BodyStructure: bs,
		Envelope:      env,
		Flags:         translateFlags(flags),
		InternalDate:  env.Date,
		RFC822Headers: &mail.Header{msg.Header},
		Size:          0,
		Uid:           m.uid,
	}, nil
}

// NewBodyPartReader creates a new io.Reader for the requested body part(s) of
// the message.
func (m Message) NewBodyPartReader(requestedParts []int) (io.Reader, error) {
	f, err := m.dir.Open(m.key)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	msg, err := message.Read(f)
	if err != nil {
		return nil, fmt.Errorf("could not read message: %v", err)
	}
	return fetchEntityPartReader(msg, requestedParts)
}

func fetchEntityPartReader(e *message.Entity, index []int) (io.Reader, error) {
	if len(index) < 1 {
		return nil, fmt.Errorf("no part to read")
	}
	if mpr := e.MultipartReader(); mpr != nil {
		idx := 0
		for {
			idx++
			part, err := mpr.NextPart()
			if err != nil {
				return nil, err
			}
			if idx == index[0] {
				rest := index[1:]
				if len(rest) < 1 {
					return fetchEntityReader(part)
				}
				return fetchEntityPartReader(part, index[1:])
			}
		}
	}
	if index[0] != 1 {
		return nil, fmt.Errorf("cannont return non-first part of non-multipart")
	}
	return fetchEntityReader(e)
}

// fetchEntityReader makes an io.Reader for the given entity. Since the
// go-message package decodes the body for us, and the UI expects to deal with
// a reader whose bytes are encoded with the part's encoding, we are in the
// interesting position of needing to re-encode the reader before sending it
// off to the UI layer.
//
// TODO: probably change the UI to expect an already-decoded reader and decode
// in the IMAP worker.
func fetchEntityReader(e *message.Entity) (io.Reader, error) {
	enc := e.Header.Get("content-transfer-encoding")
	var buf bytes.Buffer

	// base64
	if strings.EqualFold(enc, "base64") {
		wc := base64.NewEncoder(base64.StdEncoding, &buf)
		defer wc.Close()
		if _, err := io.Copy(wc, e.Body); err != nil {
			return nil, fmt.Errorf("could not base64 encode: %v", err)
		}
		return &buf, nil
	}

	// quoted-printable
	if strings.EqualFold(enc, "quoted-printable") {
		wc := quotedprintable.NewWriter(&buf)
		defer wc.Close()
		if _, err := io.Copy(wc, e.Body); err != nil {
			return nil, fmt.Errorf("could not quoted-printable encode: %v", err)
		}
		return &buf, nil
	}

	// other general encoding
	if _, err := io.Copy(&buf, e.Body); err != nil {
		return nil, err
	}

	return &buf, nil
}

// split a MIME type into its major and minor parts
func splitMIME(m string) (string, string) {
	parts := strings.Split(m, "/")
	if len(parts) != 2 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

func parseEntityStructure(e *message.Entity) (*models.BodyStructure, error) {
	var body models.BodyStructure
	contentType, ctParams, err := e.Header.ContentType()
	if err != nil {
		return nil, fmt.Errorf("could not parse content type: %v", err)
	}
	mimeType, mimeSubType := splitMIME(contentType)
	body.MIMEType = mimeType
	body.MIMESubType = mimeSubType
	body.Params = ctParams
	body.Description = e.Header.Get("content-description")
	body.Encoding = e.Header.Get("content-transfer-encoding")
	if cd := e.Header.Get("content-disposition"); cd != "" {
		contentDisposition, cdParams, err := e.Header.ContentDisposition()
		if err != nil {
			return nil, fmt.Errorf("could not parse content disposition: %v", err)
		}
		body.Disposition = contentDisposition
		body.DispositionParams = cdParams
	}
	body.Parts = []*models.BodyStructure{}
	if mpr := e.MultipartReader(); mpr != nil {
		for {
			part, err := mpr.NextPart()
			if err == io.EOF {
				return &body, nil
			} else if err != nil {
				return nil, err
			}
			ps, err := parseEntityStructure(part)
			if err != nil {
				return nil, fmt.Errorf("could not parse child entity structure: %v", err)
			}
			body.Parts = append(body.Parts, ps)
		}
	}
	return &body, nil
}

func parseEnvelope(h *mail.Header) (*models.Envelope, error) {
	date, err := h.Date()
	if err != nil {
		return nil, fmt.Errorf("could not parse date header: %v", err)
	}
	from, err := parseAddressList(h, "from")
	if err != nil {
		return nil, fmt.Errorf("could not read from address: %v", err)
	}
	to, err := parseAddressList(h, "to")
	if err != nil {
		return nil, fmt.Errorf("could not read to address: %v", err)
	}
	cc, err := parseAddressList(h, "cc")
	if err != nil {
		return nil, fmt.Errorf("could not read cc address: %v", err)
	}
	bcc, err := parseAddressList(h, "bcc")
	if err != nil {
		return nil, fmt.Errorf("could not read bcc address: %v", err)
	}
	subj, err := h.Subject()
	if err != nil {
		return nil, fmt.Errorf("could not read subject: %v", err)
	}
	msgID, err := h.Text("message-id")
	if err != nil {
		return nil, fmt.Errorf("could not read message id: %v", err)
	}
	return &models.Envelope{
		Date:      date,
		Subject:   subj,
		MessageId: msgID,
		From:      from,
		To:        to,
		Cc:        cc,
		Bcc:       bcc,
	}, nil
}

func parseAddressList(h *mail.Header, key string) ([]*models.Address, error) {
	var converted []*models.Address
	addrs, err := h.AddressList(key)
	if err != nil {
		if hdr, err := h.Text(key); err != nil && strings.Index(hdr, "@") < 0 {
			return []*models.Address{&models.Address{
				Name: hdr,
			}}, nil
		}
		return nil, err
	}
	for _, addr := range addrs {
		parts := strings.Split(addr.Address, "@")
		var mbox, host string
		if len(parts) > 1 {
			mbox = strings.Join(parts[0:len(parts)-1], "@")
			host = parts[len(parts)-1]
		} else {
			mbox = addr.Address
		}
		converted = append(converted, &models.Address{
			Name:    addr.Name,
			Mailbox: mbox,
			Host:    host,
		})
	}
	return converted, nil
}

var flagMap = map[maildir.Flag]models.Flag{
	maildir.FlagReplied: models.AnsweredFlag,
	maildir.FlagSeen:    models.SeenFlag,
	maildir.FlagTrashed: models.DeletedFlag,
	maildir.FlagFlagged: models.FlaggedFlag,
	// maildir.FlagDraft Flag = 'D'
	// maildir.FlagPassed Flag = 'P'
}

func translateFlags(maildirFlags []maildir.Flag) []models.Flag {
	var flags []models.Flag
	for _, maildirFlag := range maildirFlags {
		if flag, ok := flagMap[maildirFlag]; ok {
			flags = append(flags, flag)
		}
	}
	return flags
}
