package types

import (
	"net/textproto"
	"strings"
	"time"

	"git.sr.ht/~rjarry/aerc/models"
)

type SearchCriteria struct {
	WithFlags    models.Flags
	WithoutFlags models.Flags
	From         []string
	To           []string
	Cc           []string
	Headers      textproto.MIMEHeader
	StartDate    time.Time
	EndDate      time.Time
	SearchBody   bool
	SearchAll    bool
	Terms        string
}

func (c *SearchCriteria) PrepareHeader() {
	if c == nil {
		return
	}
	if c.Headers == nil {
		c.Headers = make(textproto.MIMEHeader)
	}
	for _, from := range c.From {
		c.Headers.Add("From", from)
	}
	for _, to := range c.To {
		c.Headers.Add("To", to)
	}
	for _, cc := range c.Cc {
		c.Headers.Add("Cc", cc)
	}
}

func (c *SearchCriteria) Combine(other *SearchCriteria) *SearchCriteria {
	if c == nil {
		return other
	}
	headers := make(textproto.MIMEHeader)
	for k, v := range c.Headers {
		headers[k] = v
	}
	for k, v := range other.Headers {
		headers[k] = v
	}
	start := c.StartDate
	if !other.StartDate.IsZero() {
		start = other.StartDate
	}
	end := c.EndDate
	if !other.EndDate.IsZero() {
		end = other.EndDate
	}
	from := make([]string, len(c.From)+len(other.From))
	copy(from[:len(c.From)], c.From)
	copy(from[len(c.From):], other.From)
	to := make([]string, len(c.To)+len(other.To))
	copy(to[:len(c.To)], c.To)
	copy(to[len(c.To):], other.To)
	cc := make([]string, len(c.Cc)+len(other.Cc))
	copy(cc[:len(c.Cc)], c.Cc)
	copy(cc[len(c.Cc):], other.Cc)
	return &SearchCriteria{
		WithFlags:    c.WithFlags | other.WithFlags,
		WithoutFlags: c.WithoutFlags | other.WithoutFlags,
		From:         from,
		To:           to,
		Cc:           cc,
		Headers:      headers,
		StartDate:    start,
		EndDate:      end,
		SearchBody:   c.SearchBody || other.SearchBody,
		SearchAll:    c.SearchAll || other.SearchAll,
		Terms:        strings.Join([]string{c.Terms, other.Terms}, " "),
	}
}
