package calendar

import (
	"bytes"
	"fmt"
	"io"
	"net/mail"
	"regexp"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
)

type Reply struct {
	MimeType     string
	Params       map[string]string
	CalendarText io.ReadWriter
	PlainText    io.ReadWriter
	Organizers   []string
}

func (cr *Reply) AddOrganizer(o string) {
	cr.Organizers = append(cr.Organizers, o)
}

// CreateReply parses a ics request and return a ics reply (RFC 2446, Section 3.2.3)
func CreateReply(reader io.Reader, from *mail.Address, partstat string) (*Reply, error) {
	cr := Reply{
		MimeType: "text/calendar",
		Params: map[string]string{
			"charset": "UTF-8",
			"method":  "REPLY",
		},
		CalendarText: &bytes.Buffer{},
		PlainText:    &bytes.Buffer{},
	}

	var (
		status ics.ParticipationStatus
		action string
	)

	switch partstat {
	case "accept":
		status = ics.ParticipationStatusAccepted
		action = "accepted"
	case "accept-tentative":
		status = ics.ParticipationStatusTentative
		action = "tentatively accepted"
	case "decline":
		status = ics.ParticipationStatusDeclined
		action = "declined"
	default:
		return nil, fmt.Errorf("participation status %s is not implemented", partstat)
	}

	name := from.Name
	if name == "" {
		name = from.Address
	}
	fmt.Fprintf(cr.PlainText, "%s has %s this invitation.", name, action)

	invite, err := parse(reader)
	if err != nil {
		return nil, err
	}

	if ok := invite.request(); !ok {
		return nil, fmt.Errorf("no reply is requested")
	}

	// update invite as a reply
	reply := invite
	reply.SetMethod(ics.MethodReply)
	reply.SetProductId("aerc")

	// check all events
	for _, vevent := range reply.Events() {
		e := event{vevent}

		// check if we should answer
		if err := e.isReplyRequested(from.Address); err != nil {
			return nil, err
		}

		// make sure we send our reply to the meeting organizer
		if organizer := e.GetProperty(ics.ComponentPropertyOrganizer); organizer != nil {
			cr.AddOrganizer(organizer.Value)
		}

		// update attendee participation status
		e.updateAttendees(status, from.Address)

		// update timestamp
		e.SetDtStampTime(time.Now())

		// remove any subcomponents of event
		e.Components = nil
	}

	// keep only timezone and event components
	reply.clean()

	if len(reply.Events()) == 0 {
		return nil, fmt.Errorf("no events to respond to")
	}

	if err := reply.SerializeTo(cr.CalendarText); err != nil {
		return nil, err
	}
	return &cr, nil
}

type calendar struct {
	*ics.Calendar
}

func parse(reader io.Reader) (*calendar, error) {
	// fix capitalized mailto for parsing of ics file
	var sb strings.Builder
	_, err := io.Copy(&sb, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to copy calendar data: %w", err)
	}
	re := regexp.MustCompile("MAILTO:(.+@)")
	str := re.ReplaceAllString(sb.String(), "mailto:${1}")

	// parse calendar
	invite, err := ics.ParseCalendar(strings.NewReader(str))
	if err != nil {
		return nil, err
	}
	return &calendar{invite}, nil
}

func (cal *calendar) request() (ok bool) {
	ok = false
	for i := range cal.CalendarProperties {
		if cal.CalendarProperties[i].IANAToken == string(ics.PropertyMethod) {
			if cal.CalendarProperties[i].Value == string(ics.MethodRequest) {
				ok = true
				return
			}
		}
	}
	return
}

func (cal *calendar) clean() {
	var clean []ics.Component
	for _, comp := range cal.Components {
		switch comp.(type) {
		case *ics.VTimezone, *ics.VEvent:
			clean = append(clean, comp)
		default:
			continue
		}
	}
	cal.Components = clean
}

type event struct {
	*ics.VEvent
}

func (e *event) isReplyRequested(from string) error {
	var present bool = false
	var rsvp bool = false
	from = strings.ToLower(from)
	for _, a := range e.Attendees() {
		if strings.ToLower(a.Email()) == from {
			present = true
			if r, ok := a.ICalParameters[string(ics.ParameterRsvp)]; ok {
				if len(r) > 0 && strings.ToLower(r[0]) == "true" {
					rsvp = true
				}
			}
		}
	}
	if !present {
		return fmt.Errorf("we are not invited")
	}
	if !rsvp {
		return fmt.Errorf("we don't have to rsvp")
	}
	return nil
}

func (e *event) updateAttendees(status ics.ParticipationStatus, from string) {
	var clean []ics.IANAProperty
	for _, prop := range e.Properties {
		if prop.IANAToken == string(ics.ComponentPropertyAttendee) {
			att := ics.Attendee{IANAProperty: prop}
			if att.Email() != from {
				continue
			}
			prop.ICalParameters[string(ics.ParameterParticipationStatus)] = []string{string(status)}
			delete(prop.ICalParameters, string(ics.ParameterRsvp))
		}
		clean = append(clean, prop)
	}
	e.Properties = clean
}
