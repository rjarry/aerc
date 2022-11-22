AERC-TEMPLATES(7)

# NAME

aerc-templates - template file specification for *aerc*(1)

# SYNOPSIS

aerc uses the go text/template package for the template parsing.
Refer to the go text/template documentation for the general syntax.

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

	- _From_: List of senders.
	- _To_: List of To recipients. Not always Available.
	- _Cc_: List of Cc recipients. Not always Available.
	- _Bcc_: List of Cc recipients. Not always Available.
	- _OriginalFrom_: List of senders of the original message.
	  Available for quoted reply and forward.

	Example:

	Get the name of the first sender.
	```
	{{(index .From 0).Name}}
	```

	Get the email address of the first sender.
	```
	{{(index .From 0).Address}}
	```

*Date and Time*
	The date and time information is always available and can be easily
	formatted.

	- _Date_: Date and time information when the compose window is opened.
	- _OriginalDate_: Date and time when the original message of received.
	  Available for quoted reply and forward.

	To format the date fields, _dateFormat_ and _toLocal_ are provided.
	Refer to the *TEMPLATE FUNCTIONS* section for details.

*Subject*
	The subject of the email is available for quoted reply and forward.

	{{.Subject}}

*MIME Type*
	MIME Type is available for quoted reply and forward.

	- _OriginalMIMEType_: MIME type info of quoted mail part. Usually
	  _text/plain_ or _text/html_.

*Original Message*
	When using quoted reply or forward, the original message is available in a
	field called _OriginalText_.

	```
	{{.OriginalText}}
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

*exec*
	Execute external command, provide the second argument to its stdin.

	```
	{{exec `/usr/local/share/aerc/filters/html` .OriginalText}}
	```

*toLocal*
	Convert the date to the local timezone as specified by the locale.

	```
	{{toLocal .Date}}
	```

*dateFormat*
	Format date and time according to the format passed as the second argument.
	The format must be specified according to go's time package format.

	```
	{{dateFormat .Date "Mon Jan 2 15:04:05 -0700 MST 2006"}}
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
	{{exec `/usr/local/share/aerc/filters/html` .OriginalText | wrap 72 | quote}}
	{{else}}
	{{wrap 72 .OriginalText | quote}}
	{{end}}
	```

# SEE ALSO

*aerc*(1) *aerc-config*(5)

# AUTHORS

Originally created by Drew DeVault <sir@cmpwn.com> and maintained by Robin
Jarry <robin@jarry.cc> who is assisted by other open source contributors. For
more information about aerc development, see https://sr.ht/~rjarry/aerc/.
