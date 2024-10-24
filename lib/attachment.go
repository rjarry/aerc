package lib

import (
	"bufio"
	"bytes"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/xdg"
	"github.com/emersion/go-message/mail"
	"github.com/pkg/errors"
)

type Part struct {
	MimeType        string
	Params          map[string]string
	Data            []byte
	Converted       bool
	ConversionError error
}

func NewPart(mimetype string, params map[string]string, body io.Reader,
) (*Part, error) {
	var d []byte
	var err error
	var converted bool
	if body == nil {
		converted = true
	} else {
		d, err = io.ReadAll(body)
		if err != nil {
			return nil, err
		}
	}
	return &Part{
		MimeType:  mimetype,
		Params:    params,
		Data:      d,
		Converted: converted,
	}, nil
}

func (p *Part) NewReader() io.Reader {
	return bytes.NewReader(p.Data)
}

type Attachment interface {
	Name() string
	WriteTo(w *mail.Writer) error
}

type FileAttachment struct {
	path string
}

func NewFileAttachment(path string) *FileAttachment {
	return &FileAttachment{
		path,
	}
}

func (fa *FileAttachment) Name() string {
	return fa.path
}

func (fa *FileAttachment) WriteTo(w *mail.Writer) error {
	f, err := os.Open(xdg.ExpandHome(fa.path))
	if err != nil {
		return errors.Wrap(err, "os.Open")
	}
	defer f.Close()

	reader := bufio.NewReader(f)

	mimeType, params, err := FindMimeType(fa.path, reader)
	if err != nil {
		return errors.Wrap(err, "ParseMediaType")
	}
	filename := filepath.Base(fa.path)
	params["name"] = filename

	// set header fields
	ah := mail.AttachmentHeader{}
	ah.SetContentType(mimeType, params)
	// setting the filename auto sets the content disposition
	ah.SetFilename(filename)

	fixContentTransferEncoding(mimeType, &ah)

	aw, err := w.CreateAttachment(ah)
	if err != nil {
		return errors.Wrap(err, "CreateAttachment")
	}
	defer aw.Close()

	if _, err := reader.WriteTo(aw); err != nil {
		return errors.Wrap(err, "reader.WriteTo")
	}

	return nil
}

type PartAttachment struct {
	part *Part
	name string
}

func NewPartAttachment(part *Part, name string) *PartAttachment {
	return &PartAttachment{
		part,
		name,
	}
}

func (pa *PartAttachment) Name() string {
	return pa.name
}

func (pa *PartAttachment) WriteTo(w *mail.Writer) error {
	// set header fields
	ah := mail.AttachmentHeader{}
	ah.SetContentType(pa.part.MimeType, pa.part.Params)

	// setting the filename auto sets the content disposition
	ah.SetFilename(pa.Name())

	fixContentTransferEncoding(pa.part.MimeType, &ah)

	aw, err := w.CreateAttachment(ah)
	if err != nil {
		return errors.Wrap(err, "CreateAttachment")
	}
	defer aw.Close()

	if _, err := io.Copy(aw, pa.part.NewReader()); err != nil {
		return errors.Wrap(err, "io.Copy")
	}
	return nil
}

// SetUtf8Charset sets the charset in a params map to UTF-8.
func SetUtf8Charset(origParams map[string]string) map[string]string {
	params := make(map[string]string)
	for k, v := range origParams {
		switch strings.ToLower(k) {
		case "charset":
			log.Debugf("substitute charset %s with utf-8", v)
			params[k] = "utf-8"
		default:
			params[k] = v
		}
	}
	return params
}

func FindMimeType(filename string, reader *bufio.Reader) (string, map[string]string, error) {
	// if we have an extension, prefer that instead of trying to sniff the header.
	// That's generally more accurate than sniffing as lots of things are zip files
	// under the hood, e.g. most office file types
	ext := filepath.Ext(filename)
	var mimeString string
	if mimeString = mime.TypeByExtension(ext); mimeString == "" {
		// Sniff the mime type since it's not in the database
		// http.DetectContentType only cares about the first 512 bytes
		head, err := reader.Peek(512)
		if err != nil && err != io.EOF {
			return "", map[string]string{}, errors.Wrap(err, "Peek")
		}
		mimeString = http.DetectContentType(head)
	}

	// mimeString can contain type and params (like text encoding),
	// so we need to break them apart before passing them to the headers
	return mime.ParseMediaType(mimeString)
}

// fixContentTransferEncoding checks the mime type of the attachment and
// corrects the content-transfer-encoding if necessary.
//
// It's expressly forbidden by RFC2046 to set any other
// content-transfer-encoding than 7bit, 8bit, or binary for
// message/rfc822 mime types (see RFC2046, section 5.2.1)
func fixContentTransferEncoding(mimeType string, header *mail.AttachmentHeader) {
	if strings.ToLower(mimeType) == "message/rfc822" {
		header.Add("Content-Transfer-Encoding", "binary")
	}
}
