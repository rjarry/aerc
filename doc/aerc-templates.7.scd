AERC-TEMPLATES(7)

# NAME

aerc-templates - template file specification for *aerc*(1)

# SYNOPSIS

aerc uses the go text/template package for the template parsing.
Refer to the go text/template documentation for the general syntax.
The template syntax described below can be used for message template files and
for dynamic formatting of some UI widgets.

Template files are composed of headers, followed by a newline, followed by the
body text.

Example:

```
X-Clacks-Overhead: GNU Terry Pratchett

Hello,

Greetings,
Chuck
```

If you have a template that doesn't add any header, it *must* be preceded by a
newline, to avoid parsing parts of the body as header text.

All headers defined in the template will have precedence over any headers that
are initialized by aerc (e.g. Subject, To, From, Cc) when composing a new
message, forwarding or replying.

# MESSAGE DATA

The following data can be used in templates. Though they are not all
available always.

*Addresses*
	An array of mail.Address. That can be used to add sender or recipient
	names to the template.

	- _{{.From}}_: List of senders.
	- _{{.Peer}}_: List of senders or To recipients if the message is from
	  you.
	- _{{.To}}_: List of To recipients. Not always Available.
	- _{{.ReplyTo}}_: List of ReplyTo recipients. Not always Available.
	- _{{.Cc}}_: List of Cc recipients. Not always Available.
	- _{{.Bcc}}_: List of Cc recipients. Not always Available.
	- _{{.OriginalFrom}}_: List of senders of the original message.
	  Available for quoted reply and forward.

	Example:

	Get the name of the first sender.
	```
	{{(index .From 0).Name}}
	{{index (.From | names) 0}}
	```

	Get the email address of the first sender.
	```
	{{(index .From 0).Address}}
	```

*Date and Time*
	The date and time information is always available and can be easily
	formatted.

	- _{{.Date}}_: Date and time information when the compose window is opened.
	- _{{.OriginalDate}}_: Date and time when the original message of received.
	  Available for quoted reply and forward.

	To format the date fields, _dateFormat_ and _.Local_ are provided.
	Refer to the *TEMPLATE FUNCTIONS* section for details.

*Subject*
	The subject of the email (_ThreadPrefix_ will be empty unless threading
	is enabled).

	```
	{{.ThreadPrefix}}{{.Subject}}
	```

	The subject of the email stripped of any _Re:_ and _Fwd:_ prefixes.

	```
	{{.SubjectBase}}
	```

*Flags*
	List of message flags, not available when composing, replying nor
	forwarding. This is a list of strings that may be converted to a single
	string with *join*.

	```
	{{.Flags | join ""}}
	```

*Labels*
	Message labels (for example notmuch tags). Not available when composing,
	replying nor forwarding. This is a list of strings that may be converted
	to a single string with *join*.

	```
	{{.Labels | join " "}}
	```

*Size*
	The size of the message in bytes. Not available when composing, replying
	nor forwarding. It can be formatted with *humanReadable*.

	```
	{{.Size | humanReadable}}
	```

*Any header value*
	Any header value of the email.

	```
	{{.Header "x-foo-bar"}}
	```

	Any header values of the original forwared or replied message:

	```
	{{.OriginalHeader "x-foo-bar"}}
	```

*MIME Type*
	MIME Type is available for quoted reply and forward.

	- _{{.OriginalMIMEType}}_: MIME type info of quoted mail part. Usually
	  _text/plain_ or _text/html_.

*Original Message*
	When using quoted reply or forward, the original message is available in a
	field called _OriginalText_.

	```
	{{.OriginalText}}
	```

*Account info*
	The current account name:

	```
	{{.Account}}
	```

	Currently selected mailbox folder:

	```
	{{.Folder}}
	```

	Current message counts for all folders:

	```
	{{.Recent}} {{.Unread}} {{.Exists}}
	{{.RUE}}
	```

	Current message counts for specific folders:

	```
	{{.Recent "inbox"}}
	{{.Unread "inbox" "aerc/pending"}}
	{{.Exists "archive" "spam" "foo/baz" "foo/bar"}}
	{{.RUE "inbox"}}
	```

*Status line*

	The following data will only be available in the status line templates:

	Connection state.

	```
	{{.Connected}}
	{{.ConnectionInfo}}
	```

	General status information (e.g. filter, search) separated with
	*[statusline].separator*.

	```
	{{.ContentInfo}}
	```

	Combination of *{{.ConnectionInfo}}* and *{{.StatusInfo}}* separated
	with *[statusline].separator*.

	```
	{{.StatusInfo}}
	```

	General on/off information (e.g. passthrough, threading, sorting),
	separated with *[statusline].separator*.

	```
	{{.TrayInfo}}
	```

	Currently pressed key sequence that does not match any key binding
	and/or is incomplete.

	```
	{{.PendingKeys}}
	```

# TEMPLATE FUNCTIONS

Besides the standard functions described in go's text/template documentation,
aerc provides the following additional functions:

*wrap*
	Wrap the original text to the specified number of characters per line.

	```
	{{wrap 72 .OriginalText}}
	```

*quote*
	Prepends each line with _"> "_.

	```
	{{quote .OriginalText}}
	```

*trimSignature*
	Removes the signature froma passed in mail. Quoted signatures are kept
	as they are.

	```
	{{trimSignature .OriginalText}}
	```

*join*
	Join the provided list of strings with a separator:

	```
	{{.To | names | join ", "}}
	```

*names*
	Extracts the names part from a mail.Address list. If there is no name
	available, the mbox (email address without @domain) is returned instead.

	```
	{{.To | names | join ", "}}
	{{index (.To | names) 0}}
	```

*firstnames*
	Extracts the first names part from a mail.Address list. If there is no
	name available, the short mbox (start of email address without @domain)
	is returned instead.

	```
	{{.To | firstnames | join ", "}}
	{{index (.To | firstnames) 0}}
	```

*initials*
	Extracts the initials from the names part from a mail.Address list. If
	there is no name available, the first letter of the email address is
	returned instead.

	```
	{{.To | initials | join ", "}}
	{{index (.To | initials) 0}}
	```

*emails*
	Extracts the addresses part from a mail.Address list.

	```
	{{.To | emails | join ", "}}
	{{index (.To | emails) 0}}
	```

*mboxes*
	Extracts the mbox part from a mail.Address list (i.e. _smith_ from
	_smith@example.com_).

	```
	{{.To | mboxes | join ", "}}
	{{index (.To | mboxes) 0}}
	```

*shortmboxes*
	Extracts the short mbox part from a mail.Address list (i.e. _smith_ from
	_smith.and.wesson@example.com_).

	```
	{{.To | shortmboxes | join ", "}}
	{{index (.To | shortmboxes) 0}}
	```

*persons*
	Formats a list of mail.Address into a list of strings containing the
	human readable form of RFC5322 (e.g. _Firstname Lastname
	<email@address.tld>_).

	```
	{{.To | persons | join ", "}}
	{{index (.To | persons) 0}}
	```

*exec*
	Execute external command, provide the second argument to its stdin.

	```
	{{exec `/usr/libexec/aerc/filters/html` .OriginalText}}
	```

*.Local*
	Convert the date to the local timezone as specified by the locale.

	```
	{{.Date.Local}}
	```

*dateFormat*
	Format date and time according to the format passed as the second argument.
	The format must be specified according to go's time package format.

	```
	{{dateFormat .Date "Mon Jan 2 15:04:05 -0700 MST 2006"}}
	```

	You can also use the _.DateAutoFormat_ method to format the date
	according to *\*-time\*format* settings:

	```
	{{.DateAutoFormat .OriginalDate.Local}}
	```

*now*
	Return the current date as a golang time.Time object that can be
	formated with *dateFormat*.

	```
	{{dateFormat now "Mon Jan 2 15:04:05 -0700 MST 2006"}}
	```

*humanReadable*
	Return the human readable form of an integer value.

	```
	{{humanReadable 3217653721}}
	```

*cwd*
	Return the current working directory with the user home dir replaced by
	_~_.

	```
	{{cwd}}
	```

*compactDir*
	Reduce a directory path into a compact form. The directory name will be
	split with _/_ and each part will be reduced to the first letter in its
	name: _INBOX/01_WORK/PROJECT_ will become _I/W/PROJECT_.

	```
	{{compactPath .Folder}}
	```

*.Style*
	Apply a user-defined style (see *aerc-stylesets*(7)) to a string.

	```
	{{.Style "red" .Account}}
	```

*version*
	Returns the version of aerc, which can be useful for things like X-Mailer.

	```
	X-Mailer: aerc {{version}}
	```

*Function chaining*
	All of the template functions can be chained together if needed.

	Example: Automatic HTML parsing for text/html mime type messages
	```
	{{if eq .OriginalMIMEType "text/html"}}
	{{exec `/usr/libexec/aerc/filters/html` .OriginalText | wrap 72 | quote}}
	{{else}}
	{{wrap 72 .OriginalText | trimSignature | quote}}
	{{end}}
	```

# SEE ALSO

*aerc*(1) *aerc-config*(5)

# AUTHORS

Originally created by Drew DeVault <sir@cmpwn.com> and maintained by Robin
Jarry <robin@jarry.cc> who is assisted by other open source contributors. For
more information about aerc development, see https://sr.ht/~rjarry/aerc/.
