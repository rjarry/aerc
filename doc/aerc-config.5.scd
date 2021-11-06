aerc-config(5)

# NAME

aerc-config - configuration file formats for *aerc*(1)

# CONFIGURATION

There are three aerc config files: *aerc.conf*, *binds.conf*, and
*accounts.conf*. The last one must be kept secret, as it may include your
account credentials. We look for these files in your XDG config home plus
"aerc", which defaults to ~/.config/aerc.

Examples of these config files are typically included with your installation of
aerc and are usually installed in /usr/share/aerc.

Each file uses the _ini_ format, and consists of sections with keys and values.
A line beginning with # is considered a comment and ignored, as are empty lines.
New sections begin with [section-name] on a single line, and keys and values are
separated with "=".

# AERC.CONF

This file is used for configuring the general appearance and behavior of aerc.

## GENERAL OPTIONS

These options are configured in the *[general]* section of aerc.conf.

*default-save-path*
	Used as a default path for save operations if no other path is specified.

## UI OPTIONS

These options are configured in the *[ui]* section of aerc.conf.

*index-format*
	Describes the format for each row in a mailbox view. This field is
	compatible with mutt's printf-like syntax.

	Default: %D %-17.17n %s

[- *Format specifier*
:[ *Description*
|  %%
:  literal %
|  %a
:  sender address
|  %A
:  reply-to address, or sender address if none
|  %C
:  message number
|  %d
:  formatted message timestamp
|  %D
:  formatted message timestamp converted to local timezone
|  %f
:  sender name and address
|  %F
:  author name, or recipient name if the message is from you.
   The address is shown if no name part.
|  %g
:  message labels (for example notmuch tags)
|  %i
:  message id
|  %n
:  sender name, or sender address if none
|  %r
:  comma-separated list of formatted recipient names and addresses
|  %R
:  comma-separated list of formatted CC names and addresses
|  %s
:  subject
|  %t
:  the (first) address the new email was sent to
|  %T
:  the account name which received the email
|  %u
:  sender mailbox name (e.g. "smith" in "smith@example.net")
|  %v
:  sender first name (e.g. "Alex" in "Alex Smith <smith@example.net>")
|  %Z
:  flags (O=old, N=new, r=answered, D=deleted, !=flagged, \*=marked)

*timestamp-format*
	See time.Time#Format at https://godoc.org/time#Time.Format

	Default: 2006-01-02 03:04 PM (ISO 8601 + 12 hour time)

*this-day-time-format*
	Index-only time format for messages that were received/sent today.
	If this is not specified, *timestamp-format* is used instead.

	Default: ""

*this-week-time-format*
	Index-only time format for messages that were received/sent within the
	last 7 days. If this is not specified, *timestamp-format* is used
	instead.

	Default: ""

*this-year-time-format*
	Index-only time format for messages that were received/sent this year.
	If this is not specified, *timestamp-format* is used instead.

	Default: ""

*sidebar-width*
	Width of the sidebar, including the border. Set to zero to disable the
	sidebar.

	Default: 20

*empty-message*
	Message to display when viewing an empty folder.

	Default: (no messages)

*empty-dirlist*
	Message to display when no folders exists or are all filtered.

	Default: (no folders)

*mouse-enabled*
	Enable mouse events in the ui, e.g. clicking and scrolling with the mousewheel

	Default: false

*new-message-bell*
	Ring the bell when a new message is received.

	Default: true

*pinned-tab-marker*
	Marker to show before a pinned tab's name.

	Default: `

*spinner*
	Animation shown while loading, split by spinner-delimiter (below)

	Examples:
	- spinner = "\-\_-,\_-\_"
	- spinner = '. , .'
	- spinner = "\,|,/,-"

	Default: "[..]    , [..]   ,  [..]  ,   [..] ,    [..],   [..] ,  [..]  , [..]   "

*spinner-delimiter*
	Spinner delimiter to split string into an animation

	Default: ","

*sort*
	List of space-separated criteria to sort the messages by, see *sort*
	command in *aerc*(1) for reference. Prefixing a criterion with "-r "
	reverses that criterion.

	Example: "from -r date"

	Default: ""

*dirlist-format*
	Describes the format string to use for the directory list

	Default: %n %>r

[- *Format specifier*
:[ *Description*
|  %%
:  literal %
|  %n
:  directory name
|  %r
:  recent/unseen/total message count
|  %>X
:  make format specifier 'X' be right justified

*next-message-on-delete*
	Moves to next message when the current message is deleted

	Default: true

*completion-popovers*
	Shows potential auto-completions for text inputs in popovers.

	Default: true

*completion-delay*
	How long to wait after the last input before auto-completion is triggered.

	Default: 250ms

*stylesets-dirs*
	The directories where the stylesets are stored. The config takes a
	colon-seperated list of dirs.

	Default: "/usr/share/aerc/stylesets"

*styleset-name*
	The name of the styleset to be used to style the ui elements. The
	stylesets are stored in the 'stylesets' directory in the config
	directory.

	Default: default

	Have a look at *aerc-stylesets*(7) as to how a styleset looks like.


## Contextual UI Configuration

The UI configuration can be specialized for accounts, specific mail
directories and message subjects. The specializations are added using
contextual config sections based on the context.

The contextual UI configuration is merged to the base UiConfig in the
following order:
*Base UIConfig > Account Context > Folder Context > Subject Context.*

*[ui:account=<AccountName>]*
	Adds account specific configuration with the account name.

*[ui:folder=<FolderName>]*
	Add folder specific configuration with the folder name.

*[ui:folder~<Regex>]*
	Add folder specific configuration for folders whose names match the regular
	expression.

*[ui:subject~<Regex>]*
	Add specialized ui configuration for messages that match a given regular
	expression.

Example:
```
[ui:account=Work]
sidebar-width=...

[ui:folder=Sent]
index-format=...

[ui:folder~Archive/\d+/.*]
index-format=...

[ui:subject~^\[PATCH]
index-format=...
```


## VIEWER

These options are configured in the *[viewer]* section of aerc.conf.

*pager*
	Specifies the pager to use when displaying emails. Note that some filters
	may add ANSI escape sequences to add color to rendered emails, so you may
	want to use a pager which supports ANSI.

	Default: less -R

*alternatives*
	If an email offers several versions (multipart), you can configure which
	mimetype to prefer. For example, this can be used to prefer plaintext over
	HTML emails.

	Default: text/plain,text/html

*header-layout*
	Defines the default headers to display when viewing a message. To display
	multiple headers in the same row, separate them with a pipe, e.g. "From|To".
	Rows will be hidden if none of their specified headers are present in the
	message.

	Default: From|To,Cc|Bcc,Date,Subject

*show-headers*
	Default setting to determine whether to show full headers or only parsed
	ones in message viewer.

	Default: false

*always-show-mime*
	Whether to always show the mimetype of an email, even when it is just a single part.

	Default: false

## COMPOSE

These options are configured in the *[compose]* section of aerc.conf.

*editor*
	Specifies the command to run the editor with. It will be shown in an
	embedded terminal, though it may also launch a graphical window if the
	environment supports it. Defaults to *$EDITOR*, or *vi*(1).

*header-layout*
	Defines the default headers to display when composing a message. To display
	multiple headers in the same row, separate them with a pipe, e.g. "To|From".

	Default: To|From,Subject

*address-book-cmd*
	Specifies the command to be used to tab-complete email addresses. Any
	occurrence of "%s" in the address-book-cmd will be replaced with what the
	user has typed so far.

	The command must output the completions to standard output, one completion
	per line. Each line must be tab-delimited, with an email address occurring as
	the first field. Only the email address field is required. The second field,
	if present, will be treated as the contact name. Additional fields are
	ignored.

	Example:
		khard email --remove-first-line --parsable '%s'

	Default: none

## FILTERS

Filters allow you to pipe an email body through a shell command to render
certain emails differently, e.g. highlighting them with ANSI escape codes.
They are configured in the *[filters]* section of aerc.conf.

The first filter which matches the email's mimetype will be used, so order
them from most to least specific.

You can also match on non-mimetypes, by prefixing with the header to match
against (non-case-sensitive) and a comma, e.g. subject,text will match a
subject which contains "text". Use header,~regex to match against a regex.

aerc ships with some default filters installed in the share directory (usually
_/usr/share/aerc/filters_). Note that these may have additional dependencies
that aerc does not have alone.

## TRIGGERS

Triggers specify commands to execute when certain events occur.

They are configured in the *[triggers]* section of aerc.conf.

*new-email*
	Executed when a new email arrives in the selected folder.

	e.g. new-email=exec notify-send "New email from %n" "%s"

	Default: ""

	Format specifiers from *index-format* are expanded with respect to the new
	message.

## TEMPLATES

Templates are used to populate the body of an email. The compose, reply
and forward commands can be called with the -T flag with the name of the
template name.

aerc ships with some default templates installed in the share directory (usually
_/usr/share/aerc/templates_).

These options are configured in the *[templates]* section of aerc.conf.

*template-dirs*
	The directory where the templates are stored. The config takes a
	colon-separated list of dirs.

	Default: "/usr/share/aerc/templates"

*quoted-reply*
	The template to be used for quoted replies.

	Default: "quoted_reply"

*forwards*
	The template to be used for forward as body.

	Default: "forward_as_body"

# ACCOUNTS.CONF

This file is used for configuring each mail account used for aerc. Each section
is the name of an account you want to configure, and the keys & values in that
section specify details of that account's configuration. In addition to the
options documented here, specific transports for incoming and outgoing emails
may have additional configuration parameters, documented on their respective man
pages.

Note that many of these configuration options are written for you, such as
*source* and *outgoing*, when you run the account configuration wizard
(*:new-account*).

*archive*
	Specifies a folder to use as the destination of the *:archive* command.

	Default: Archive

*copy-to*
	Specifies a folder to copy sent mails to, usually "Sent".

	Default: none

*default*
	Specifies the default folder to open in the message list when aerc
	configures this account.

	Default: INBOX

*folders*
	Specifies the comma separated list of folders to display in the sidebar.
	Names prefixed with ~ are interpreted as regular expressions.

	Default: all folders

*folders-exclude*
	Specifies the comma separated list of folders to exclude from the sidebar.
	Names prefixed with ~ are interpreted as regular expressions.
	Note that this overrides anything from *folders*.

	Default: no folders

*folders-sort*
	Specifies a comma separated list of folders to be shown at the top of the
	list in the provided order. Remaining folders will be sorted alphabetically.

	Default: none

*from*
	The default value to use for the From header in new emails. This should be
	an RFC 5322-compatible string, such as "Your Name <you@example.org>".

	Default: none

*aliases*
	All aliases of the current account. These will be used to fill in the From:
	field. Make sure that your email server accepts this value, or for example
	use *aerc-sendmail*(5) in combination with msmtp and --read-envelope-from.

	Default: none

*outgoing*
	Specifies the transport for sending outgoing emails on this account.  It
	should be a connection string, and the specific meaning of each component
	varies depending on the protocol in use.  See each protocol's man page for
	more details:

	- *aerc-smtp*(5)

*outgoing-cred-cmd*
	Specifies an optional command that is run to get the outgoing account's
	password. See each protocol's man page for more details.

	Default: none

*postpone*
	Specifies the folder to save postponed messages to.

	Default: Drafts

*source*
	Specifies the source for reading incoming emails on this account. This key
	is required for all accounts. It should be a connection string, and the
	specific meaning of each component varies depending on the protocol in use.
	See each protocol's man page for more details:

	- *aerc-imap*(5)
	- *aerc-maildir*(5)
	- *aerc-notmuch*(5)

	Default: none

*source-cred-cmd*
	Specifies an optional command that is run to get the source account's
	password. See each protocol's man page for more details.

*signature-file*
	Specifies the file to read in order to obtain the signature to be added
	to emails sent from this account.

*signature-cmd*
	Specifies the command to execute in *sh* in order to obtain the
	signature to be added to emails sent from this account. If the command
	fails then *signature-file* is used instead.

# BINDS.CONF

This file is used for configuring keybindings used in the aerc interactive
client. You may configure different keybindings for different contexts by
writing them into different *[sections]* of the ini file. The available contexts
are:

*[messages]*
	keybindings for the message list

*[view]*
	keybindings for the message viewer

*[compose]*
	keybindings for the message composer

*[compose::editor]*
	keybindings for the composer, when the editor is focused

*[compose::review]*
	keybindings for the composer, when reviewing the email before it's sent

*[terminal]*
	keybindings for terminal tabs

You may also configure global keybindings by placing them at the beginning of
the file, before specifying any context-specific sections. For each *key=value*
option specified, the _key_ is the keystrokes pressed (in order) to invoke this
keybinding, and _value_ specifies keystrokes that aerc will simulate when the
keybinding is invoked. Generally this is used to execute commands, for example:

	rq = :reply -q<Enter>

Pressing r, then q, will simulate typing in ":reply -q<Enter>", and execute
:reply -q accordingly. It is also possible to invoke keybindings recursively in
a similar fashion. Additionally, the following special options are available in
each binding context:

*$noinherit*
	If set to "true", global keybindings will not be effective in this context.

	Default: false

*$ex*
	This can be set to a keystroke which will bring up the command input in this
	context.
	
	Default: <semicolon>

In addition to letters, special keys may be specified in <angle brackets>. The
following special keys are supported:

[[ *Name*
:- *Description*
|  space
:  " "
|  semicolon
:  ;
|  tab
:  
|  enter
:  
|  up
:  
|  c-up
:  Ctrl+Up
|  down
:  
|  c-down
:  Ctrl+Down
|  right
:  
|  c-right
:  Ctrl+Right
|  left
:  
|  c-left
:  Ctrl+Left
|  pgup
:  
|  c-pgup
:  Ctrl+PageUp
|  pgdn
:  
|  c-pgdn
:  Ctrl+PageUp
|  home
:  
|  end
:  
|  insert
:  
|  delete
:  
|  exit
:  
|  cancel
:  
|  print
:  
|  pause
:  
|  backtab
: 
|  c-space
:  Ctrl+Space
|  c-a
:  Ctrl+a
|  c-b
:  Ctrl+b
|  c-c
:  Ctrl+c
|  c-d
:  Ctrl+d
|  c-e
:  Ctrl+e
|  c-f
:  Ctrl+f
|  c-g
:  Ctrl+g
|  c-h
:  Ctrl+h
|  c-i
:  Ctrl+i
|  c-j
:  Ctrl+j
|  c-k
:  Ctrl+k
|  c-l
:  Ctrl+l
|  c-m
:  Ctrl+m
|  c-n
:  Ctrl+n
|  c-o
:  Ctrl+o
|  c-p
:  Ctrl+p
|  c-q
:  Ctrl+q
|  c-r
:  Ctrl+r
|  c-s
:  Ctrl+s
|  c-t
:  Ctrl+t
|  c-u
:  Ctrl+u
|  c-v
:  Ctrl+v
|  c-w
:  Ctrl+w
|  c-x
:  Ctrl+x
|  c-y
:  Ctrl+y
|  c-z
:  Ctrl+z
|  c-]
:  Ctrl+]
|  c-[
:  Ctrl+[
|  c-^
:  Ctrl+^
|  c-\_
:  Ctrl+\_

# SEE ALSO

*aerc*(1) *aerc-imap*(5) *aerc-smtp*(5) *aerc-maildir*(5) *aerc-sendmail*(5)
*aerc-notmuch*(5) *aerc-stylesets*(7)

# AUTHORS

Maintained by Drew DeVault <sir@cmpwn.com>, who is assisted by other open
source contributors. For more information about aerc development, see
https://git.sr.ht/~rjarry/aerc.
