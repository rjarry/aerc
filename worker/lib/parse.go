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

var DateParseError = errors.New("date parsing failed")

func parseEnvelope(h *mail.Header) (*models.Envelope, error) {
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
	date, err := parseDate(h)
	if err != nil {
		// still return a valid struct plus a sentinel date parsing error
		// if only the date parsing failed
		err = fmt.Errorf("%w: %v", DateParseError, err)
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
	}, err
}

// parseDate tries to parse the date from the Date header with non std formats
// if this fails it tries to parse the received header as well
func parseDate(h *mail.Header) (time.Time, error) {
	t, err := h.Date()
	if err == nil {
		return t, nil
	}
	text, err := h.Text("date")
	// sometimes, no error occurs but the date is empty.
	// In this case, guess time from received header field
	if err != nil || text == "" {
		t, err := parseReceivedHeader(h)
		if err == nil {
			return t, nil
		}
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
	// still no success, try the received header as a last resort
	t, err = parseReceivedHeader(h)
	if err != nil {
		return time.Time{}, fmt.Errorf("unrecognized date format: %s", text)
	}
	return t, nil
}

func parseReceivedHeader(h *mail.Header) (time.Time, error) {
	guess, err := h.Text("received")
	if err != nil {
		return time.Time{}, fmt.Errorf("received header not parseable: %v",
			err)
	}
	return time.Parse(time.RFC1123Z, dateRe.FindString(guess))
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
		converted = append(converted, &models.Address{
			Name:    addr.Name,
			Address: addr.Address,
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
	h := &mail.Header{msg.Header}
	env, err := parseEnvelope(h)
	if err != nil && !errors.Is(err, DateParseError) {
		return nil, fmt.Errorf("could not parse envelope: %v", err)
		// if only the date parsing failed we still get the rest of the
		// envelop structure in a valid state.
		// Date parsing errors are fairly common and it's better to be
		// slightly off than to not be able to read the mails at all
		// hence we continue here
	}
	recDate, _ := parseReceivedHeader(h)
	if recDate.IsZero() {
		// better than nothing, if incorrect
		recDate = env.Date
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
		InternalDate:  recDate,
		RFC822Headers: &mail.Header{msg.Header},
		Size:          0,
		Uid:           raw.UID(),
	}, nil
}
