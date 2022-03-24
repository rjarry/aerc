#!/usr/bin/env python3

"""Parse a vcard file given via stdin and output some details.
Currently the following details are displayed if present:

- start date and time
- the summary information of the event
- a list of attendees
- the description of the event

Please note: if multiple events are included in the data then only the
first one will be parsed and displayed!

REQUIREMENTS:
- Python 3
- Python 3 - vobject library

To use as a filter in aerc, add the following line to your aerc.config:
text/calendar=show-ics-details.py
"""

import re
import sys

import vobject


def remove_mailto(message: str) -> str:
    """Remove a possible existing 'mailto:' from the given message.

    Keyword arguments:
    message -- A message string.
    """
    return re.sub(r'^mailto:', '', message, flags=re.IGNORECASE)

def extract_field(cal: vobject.icalendar.VCalendar2_0, name: str) -> str:
    """Extract the desired field from the given calendar object.

    Keyword arguments:
    cal -- A VCalendar 2.0 object.
    name -- The field name.
    """
    try:
        name = name.strip()
        if name == 'attendees':
            attendees = []
            for attendee in cal.vevent.attendee_list:
                attendees.append(remove_mailto(attendee.valueRepr()).strip())
            return ', '.join(attendees)
        elif name == 'description':
            return cal.vevent.description.valueRepr().strip()
        elif name == 'dtstart':
            return str(cal.vevent.dtstart.valueRepr()).strip()
        elif name == 'organizer':
            return remove_mailto(cal.vevent.organizer.valueRepr()).strip()
        elif name == 'summary':
            return cal.vevent.summary.valueRepr().strip()
        else:
            return ''
    except AttributeError:
        return ''

attendees   = ''
description = ''
dtstart     = ''
error       = ''
organizer   = ''
summary     = ''

try:
    cal         = vobject.readOne(sys.stdin)
    attendees   = extract_field(cal, 'attendees')
    description = extract_field(cal, 'description')
    dtstart     = extract_field(cal, 'dtstart')
    organizer   = extract_field(cal, 'organizer')
    summary     = extract_field(cal, 'summary')
except vobject.base.ParseError:
    error = '**Sorry, but we could not parse the calendar!**'

if error:
    print(error)
    print("")

print(f"Date/Time : {dtstart}")
print(f"Summary   : {summary}")
print(f"Organizer : {organizer}")
print(f"Attendees : {attendees}")
print("")
print(description)
