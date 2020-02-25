aerc-templates(7)

# NAME

aerc-templates - template file specification for *aerc*(1)

# SYNOPSIS

aerc uses the go "text/template" package for the template parsing
which supports basic go lang operations.

# MESSAGE DATA

The following data can be used in templates. Though they are not all
available always.

*Addresses*
	An array of mail.Address. That can be used to add sender or recipient
	names to the template.

	- From: List of senders.
	- To: List of To recipients. Not always Available.
	- Cc: List of Cc recipients. Not always Available.
	- Bcc: List of Cc recipients. Not always Available.
	- OriginalFrom: List of senders of the original message.
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
	formated.

	- Date: Date and Time information when the compose window is opened.
	- OriginalDate: Date and Time when the original message of received.
	  Available for quoted reply and forward.

	The _dateFormat_ function can be used to format the date and time.

	The _toLocal_ function converts a time to the local time zone.

	Example:

	Format the date to go's time package format options.
	```
	{{dateFormat .Date "Mon Jan 2 15:04:05 -0700 MST 2006"}}
	```

	Format the date in the local time zone.
	```
	{{dateFormat (.Date | toLocal) "Mon Jan 2 15:04:05 -0700 MST 2006"}}
	```

*Subject*
	The subject of the email is available for quoted reply and forward.

	Example:
	{{.Subject}}

*MIME Type*
	MIME Type is available for quoted reply.

	- OriginalMIMEType: MIME type info of quoted mail part. Usually
	  "text/plain" or "text/html".

*Original Message*
	When using quoted reply or forward, the original message is available.
	It can be used using two functions that are available to templates.

	Example:

	_wrap_ function can be used to wrap the original text to a number
	of characters per line.
	```
	{{wrap 72 .OriginalText}}
	```

	_quote_ function prepends each line with "> ".
	```
	{{quote .OriginalText}}
	```

	_exec_ function execute external command to process message.
	```
	{{exec `/usr/local/share/aerc/filters/html`}}
	```

	All of the above can be chained together if needed, for example.
	```
	{{exec `/usr/local/share/aerc/filters/html` .OriginalText | wrap 72 | quote}}
	```

	Automatic HTML parsing can be achieved.
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

Maintained by Drew DeVault <sir@cmpwn.com>, who is assisted by other open
source contributors. For more information about aerc development, see
https://git.sr.ht/~sircmpwn/aerc.
