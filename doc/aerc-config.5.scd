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

## UI OPTIONS

These options are configured in the *[ui]* section of aerc.conf.

*index-format*
	Describes the format for each row in a mailbox view. This field is
	compatible with mutt's printf-like syntax. TODO: document properly

	Default: %D %-17.17n %s

*timestamp-format*
	See time.Time#Format at https://godoc.org/time#Time.Format

	Default: 2006-01-02 03:04 PM (ISO 8601 + 12 hour time)

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

## COMPOSE

These options are configured in the *[viewer]* section of aerc.conf.

*editor*
	Specifies the command to run the editor with. It will be shown in an
	embedded terminal, though it may also launch a graphical window if the
	environment supports it. Defaults to *$EDITOR*, or *vi*(1).

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
	Specifies the list of folders to display in the sidebar.

	Default: all folders

*from*
	The default value to use for the From header in new emails. This should be
	an RFC 5322-compatible string, such as "Your Name <you@example.org>".

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

*source*
	Specifies the source for reading incoming emails on this account. This key
	is required for all accounts. It should be a connection string, and the
	specific meaning of each component varies depending on the protocol in use.
	See each protocol's man page for more details:

	- *aerc-imap*(5)

	Default: none

*source-cred-cmd*
	Specifies an optional command that is run to get the source account's
	password. See each protocol's man page for more details.


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
|  down
:  
|  right
:  
|  left
:  
|  pgup
:  
|  pgdn
:  
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
|  c-_
:  Ctrl+_

# SEE ALSO

*aerc*(1) *aerc-imap*(5) *aerc-smtp*(5)

# AUTHORS

Maintained by Drew DeVault <sir@cmpwn.com>, who is assisted by other open
source contributors. For more information about aerc development, see
https://git.sr.ht/~sircmpwn/aerc.
