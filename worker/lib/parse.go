package lib

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"git.sr.ht/~sircmpwn/aerc/models"
	"github.com/emersion/go-message"
	_ "github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
)

// RFC 1123Z regexp
var dateRe = regexp.MustCompile(`(((Mon|Tue|Wed|Thu|Fri|Sat|Sun))[,]?\s[0-9]{1,2})\s` +
	`(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s` +
	`([0-9]{4})\s([0-9]{2}):([0-9]{2})(:([0-9]{2}))?\s([\+|\-][0-9]{4})\s?`)

func FetchEntityPartReader(e *message.Entity, index []int) (io.Reader, error) {
	if len(index) == 0 {
		// non multipart, simply return everything
		return bufReader(e)
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
					return bufReader(part)
				}
				return FetchEntityPartReader(part, index[1:])
			}
		}
	}
	return nil, fmt.Errorf("FetchEntityPartReader: unexpected code reached")
}

//TODO: the UI doesn't seem to like readers which aren't buffers
func bufReader(e *message.Entity) (io.Reader, error) {
	var buf bytes.Buffer
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

func ParseEntityStructure(e *message.Entity) (*models.BodyStructure, error) {
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
			ps, err := ParseEntityStructure(part)
			if err != nil {
				return nil, fmt.Errorf("could not parse child entity structure: %v", err)
			}
			body.Parts = append(body.Parts, ps)
		}
	}
	return &body, nil
}

func parseEnvelope(h *mail.Header) (*models.Envelope, error) {
	date, err := parseDate(h)
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
	replyTo, err := parseAddressList(h, "reply-to")
	if err != nil {
		return nil, fmt.Errorf("could not read reply-to address: %v", err)
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
		ReplyTo:   replyTo,
		To:        to,
		Cc:        cc,
		Bcc:       bcc,
	}, nil
}

// parseDate extends the built-in date parser with additional layouts which are
// non-conforming but appear in the wild.
func parseDate(h *mail.Header) (time.Time, error) {
	t, parseErr := h.Date()
	if parseErr == nil {
		return t, nil
	}
	text, err := h.Text("date")
	if err != nil {
		return time.Time{}, errors.New("no date header")
	}
	// sometimes, no error occurs but the date is empty. In this case, guess time from received header field
	if text == "" {
		guess, err := h.Text("received")
		if err != nil {
			return time.Time{}, errors.New("no received header")
		}
		t, _ := time.Parse(time.RFC1123Z, dateRe.FindString(guess))
		return t, nil
	}
	layouts := []string{
		// X-Mailer: EarthLink Zoo Mail 1.0
		"Mon, _2 Jan 2006 15:04:05 -0700 (GMT-07:00)",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, text); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized date format: %s", text)
}

func parseAddressList(h *mail.Header, key string) ([]*models.Address, error) {
	var converted []*models.Address
	addrs, err := h.AddressList(key)
	if err != nil {
		if hdr, err := h.Text(key); err == nil {
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

// RawMessage is an interface that describes a raw message
type RawMessage interface {
	NewReader() (io.Reader, error)
	ModelFlags() ([]models.Flag, error)
	Labels() ([]string, error)
	UID() uint32
}

// MessageInfo populates a models.MessageInfo struct for the message.
// based on the reader returned by NewReader
func MessageInfo(raw RawMessage) (*models.MessageInfo, error) {
	r, err := raw.NewReader()
	if err != nil {
		return nil, err
	}
	msg, err := message.Read(r)
	if err != nil {
		return nil, fmt.Errorf("could not read message: %v", err)
	}
	bs, err := ParseEntityStructure(msg)
	if err != nil {
		return nil, fmt.Errorf("could not get structure: %v", err)
	}
	env, err := parseEnvelope(&mail.Header{msg.Header})
	if err != nil {
		return nil, fmt.Errorf("could not get envelope: %v", err)
	}
	flags, err := raw.ModelFlags()
	if err != nil {
		return nil, err
	}
	labels, err := raw.Labels()
	if err != nil {
		return nil, err
	}
	return &models.MessageInfo{
		BodyStructure: bs,
		Envelope:      env,
		Flags:         flags,
		Labels:        labels,
		InternalDate:  env.Date,
		RFC822Headers: &mail.Header{msg.Header},
		Size:          0,
		Uid:           raw.UID(),
	}, nil
}
