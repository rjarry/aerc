package lib

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"git.sr.ht/~rjarry/aerc/lib/parse"
	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
	"github.com/emersion/go-message"
	_ "github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
)

// RFC 1123Z regexp
var dateRe = regexp.MustCompile(`(((Mon|Tue|Wed|Thu|Fri|Sat|Sun))[,]?\s[0-9]{1,2})\s` +
	`(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s` +
	`([0-9]{4})\s([0-9]{2}):([0-9]{2})(:([0-9]{2}))?\s([\+|\-][0-9]{4})`)

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

// TODO: the UI doesn't seem to like readers which aren't buffers
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

func fixContentType(h message.Header) (string, map[string]string) {
	ct, rest := h.Get("Content-Type"), ""
	if i := strings.Index(ct, ";"); i > 0 {
		ct, rest = ct[:i], ct[i:]
	}

	// check if there are quotes around the content type
	if strings.Contains(ct, "\"") {
		header := strings.ReplaceAll(ct, "\"", "")
		if rest != "" {
			header += rest
		}
		h.Set("Content-Type", header)
		if contenttype, params, err := h.ContentType(); err == nil {
			return contenttype, params
		}
	}

	// if all else fails, return text/plain
	return "text/plain", nil
}

func ParseEntityStructure(e *message.Entity) (*models.BodyStructure, error) {
	var body models.BodyStructure
	contentType, ctParams, err := e.Header.ContentType()
	if err != nil {
		// try to fix the error; if all measures fail, then return a
		// text/plain content type to display at least plaintext
		contentType, ctParams = fixContentType(e.Header)
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
			return nil, fmt.Errorf("could not parse content disposition: %w", err)
		}
		body.Disposition = contentDisposition
		body.DispositionParams = cdParams
	}
	body.Parts = []*models.BodyStructure{}
	if mpr := e.MultipartReader(); mpr != nil {
		for {
			part, err := mpr.NextPart()
			if errors.Is(err, io.EOF) {
				return &body, nil
			} else if err != nil {
				return nil, err
			}
			ps, err := ParseEntityStructure(part)
			if err != nil {
				return nil, fmt.Errorf("could not parse child entity structure: %w", err)
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
		return nil, fmt.Errorf("could not read from address: %w", err)
	}
	to, err := parseAddressList(h, "to")
	if err != nil {
		return nil, fmt.Errorf("could not read to address: %w", err)
	}
	cc, err := parseAddressList(h, "cc")
	if err != nil {
		return nil, fmt.Errorf("could not read cc address: %w", err)
	}
	bcc, err := parseAddressList(h, "bcc")
	if err != nil {
		return nil, fmt.Errorf("could not read bcc address: %w", err)
	}
	replyTo, err := parseAddressList(h, "reply-to")
	if err != nil {
		return nil, fmt.Errorf("could not read reply-to address: %w", err)
	}
	subj, err := h.Subject()
	if err != nil {
		return nil, fmt.Errorf("could not read subject: %w", err)
	}
	msgID, err := h.MessageID()
	if err != nil {
		// proper parsing failed, so fall back to whatever is there
		msgID, err = h.Text("message-id")
		if err != nil {
			return nil, err
		}
	}
	var irt string
	irtList := parse.MsgIDList(h, "in-reply-to")
	if len(irtList) > 0 {
		irt = irtList[0]
	}
	date, err := parseDate(h)
	if err != nil {
		// still return a valid struct plus a sentinel date parsing error
		// if only the date parsing failed
		err = fmt.Errorf("%w: %v", DateParseError, err) //nolint:errorlint // can only use %w once
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
		InReplyTo: irt,
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
		return time.Time{}, fmt.Errorf("received header not parseable: %w",
			err)
	}
	return time.Parse(time.RFC1123Z, dateRe.FindString(guess))
}

func parseAddressList(h *mail.Header, key string) ([]*mail.Address, error) {
	hdr, err := h.Text(key)
	if err != nil && !message.IsUnknownCharset(err) {
		return nil, err
	}
	if hdr == "" {
		return nil, nil
	}
	add, err := mail.ParseAddressList(hdr)
	if err != nil {
		return []*mail.Address{{Name: hdr}}, nil
	}
	return add, err
}

// RawMessage is an interface that describes a raw message
type RawMessage interface {
	NewReader() (io.ReadCloser, error)
	ModelFlags() (models.Flags, error)
	Labels() ([]string, error)
	UID() uint32
}

// MessageInfo populates a models.MessageInfo struct for the message.
// based on the reader returned by NewReader
func MessageInfo(raw RawMessage) (*models.MessageInfo, error) {
	var parseErr error
	r, err := raw.NewReader()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	msg, err := ReadMessage(r)
	if err != nil {
		return nil, fmt.Errorf("could not read message: %w", err)
	}
	bs, err := ParseEntityStructure(msg)
	if errors.As(err, new(message.UnknownEncodingError)) {
		parseErr = err
	} else if err != nil {
		return nil, fmt.Errorf("could not get structure: %w", err)
	}
	h := &mail.Header{Header: msg.Header}
	env, err := parseEnvelope(h)
	if err != nil && !errors.Is(err, DateParseError) {
		return nil, fmt.Errorf("could not parse envelope: %w", err)
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
		RFC822Headers: &mail.Header{Header: msg.Header},
		Size:          0,
		Uid:           raw.UID(),
		Error:         parseErr,
	}, nil
}

// LimitHeaders returns a new Header with the specified headers included or
// excluded
func LimitHeaders(hdr *mail.Header, fields []string, exclude bool) {
	fieldMap := make(map[string]struct{}, len(fields))
	for _, f := range fields {
		fieldMap[strings.ToLower(f)] = struct{}{}
	}
	curFields := hdr.Fields()
	for curFields.Next() {
		key := strings.ToLower(curFields.Key())
		_, ok := fieldMap[key]
		switch {
		case exclude && ok:
			curFields.Del()
		case exclude && !ok:
			// No-op: exclude but we didn't find it
		case !exclude && ok:
			// No-op: include and we found it
		case !exclude && !ok:
			curFields.Del()
		}
	}
}

// MessageHeaders populates a models.MessageInfo struct for the message.
// based on the reader returned by NewReader. Minimal information is included.
// There is no body structure or RFC822Headers set
func MessageHeaders(raw RawMessage) (*models.MessageInfo, error) {
	var parseErr error
	r, err := raw.NewReader()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	msg, err := ReadMessage(r)
	if err != nil {
		return nil, fmt.Errorf("could not read message: %w", err)
	}
	h := &mail.Header{Header: msg.Header}
	env, err := parseEnvelope(h)
	if err != nil && !errors.Is(err, DateParseError) {
		return nil, fmt.Errorf("could not parse envelope: %w", err)
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
		Envelope:     env,
		Flags:        flags,
		Labels:       labels,
		InternalDate: recDate,
		Refs:         parse.MsgIDList(h, "references"),
		Size:         0,
		Uid:          raw.UID(),
		Error:        parseErr,
	}, nil
}

// NewCRLFReader returns a reader with CRLF line endings
func NewCRLFReader(r io.Reader) io.Reader {
	var buf bytes.Buffer
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		buf.WriteString(scanner.Text() + "\r\n")
	}
	return &buf
}

// ReadMessage is a wrapper for the message.Read function to read a message
// from r. The message's encoding and charset are automatically decoded to
// UTF-8. If an unknown charset is encountered, the error is logged but a nil
// error is returned since the entity object can still be read.
func ReadMessage(r io.Reader) (*message.Entity, error) {
	entity, err := message.Read(r)
	if message.IsUnknownCharset(err) {
		log.Warnf("unknown charset encountered")
	} else if err != nil {
		return nil, fmt.Errorf("could not read message: %w", err)
	}
	return entity, nil
}

// FileSize returns the size of the file specified by name
func FileSize(name string) (uint32, error) {
	fileInfo, err := os.Stat(name)
	if err != nil {
		return 0, fmt.Errorf("failed to obtain fileinfo: %w", err)
	}
	return uint32(fileInfo.Size()), nil
}
