AERC-CONFIG(5)

# NAME

aerc-config - configuration file formats for *aerc*(1)

# CONFIGURATION

There are three aerc config files: _aerc.conf_, _binds.conf_, and
_accounts.conf_. The last one must be kept secret, as it may include your
account credentials. We look for these files in your XDG config home plus
_aerc_, which defaults to _~/.config/aerc_.

Examples of these config files are typically included with your installation of
aerc and are usually installed in _/usr/share/aerc_.

Each file uses the ini format, and consists of sections with keys and values.
A line beginning with _#_ is considered a comment and ignored, as are empty
lines. New sections begin with _[section-name]_ on a single line, and keys and
values are separated with _=_.

# AERC.CONF

This file is used for configuring the general appearance and behavior of aerc.

## GENERAL OPTIONS

These options are configured in the *[general]* section of _aerc.conf_.

*default-save-path*
	Used as a default path for save operations if no other path is specified.

*pgp-provider*
	If set to _gpg_, aerc will use system gpg binary and keystore for all
	crypto operations. Otherwise, the internal openpgp implementation will
	be used.

	Default: _internal_

*unsafe-accounts-conf*
	By default, the file permissions of _accounts.conf_ must be restrictive
	and only allow reading by the file owner (_0600_). Set this option to
	_true_ to ignore this permission check. Use this with care as it may
	expose your credentials.

	Default: _false_

*log-file*
	Output log messages to specified file. A path starting with _~/_ is
	expanded to the user home dir. When redirecting aerc's output to a file
	using _>_ shell redirection, this setting is ignored and log messages
	are printed to stdout.

*log-level*
	Only log messages above the specified level to *log-file*. Supported
	levels are: _debug_, _info_, _warn_ and _error_. When redirecting aerc's
	output to a file using _>_ shell redirection, this setting is ignored
	and the log level is forced to _debug_.

	Default: _info_

## UI OPTIONS

These options are configured in the *[ui]* section of _aerc.conf_.

*index-format*
	Describes the format for each row in a mailbox view. This field is
	compatible with mutt's printf-like syntax.

	Default: _%D %-17.17n %s_

[- *Format specifier*
:[ *Description*
|  _%%_
:  literal %
|  _%a_
:  sender address
|  _%A_
:  reply-to address, or sender address if none
|  _%C_
:  message number
|  _%d_
:  formatted message timestamp
|  _%D_
:  formatted message timestamp converted to local timezone
|  _%f_
:  sender name and address
|  _%F_
:  author name, or recipient name if the message is from you.
   The address is shown if no name part.
|  _%g_
:  message labels (for example notmuch tags)
|  _%i_
:  message id
|  _%n_
:  sender name, or sender address if none
|  _%r_
:  comma-separated list of formatted recipient names and addresses
|  _%R_
:  comma-separated list of formatted CC names and addresses
|  _%s_
:  subject
|  _%t_
:  the (first) address the new email was sent to
|  _%T_
:  the account name which received the email
|  _%u_
:  sender mailbox name (e.g. "smith" in "smith@example.net")
|  _%v_
:  sender first name (e.g. "Alex" in "Alex Smith <smith@example.net>")
|  _%Z_
:  flags (O=old, N=new, r=answered, D=deleted, !=flagged, \*=marked)

*timestamp-format*
	See time.Time#Format at https://godoc.org/time#Time.Format

	Default: _2006-01-02 03:04 PM_ (ISO 8601 + 12 hour time)

*this-day-time-format*
	Index-only time format for messages that were received/sent today.
	If this is not specified, *timestamp-format* is used instead.

*this-week-time-format*
	Index-only time format for messages that were received/sent within the
	last 7 days. If this is not specified, *timestamp-format* is used
	instead.

*this-year-time-format*
	Index-only time format for messages that were received/sent this year.
	If this is not specified, *timestamp-format* is used instead.

*message-view-timestamp-format*
	If set, overrides *timestamp-format* for the message view.

*message-view-this-day-time-format*
	If set, overrides *timestamp-format* in the message view for messages
	that were received/sent today.

*message-view-this-week-time-format*
	If set, overrides *timestamp-format* in the message view for messages
	that were recieved/sent within the last 7 days.

*message-view-this-year-time-format*
	If set, overrides *timestamp-format* in the message view for messages
	that were received/sent this year.

*sidebar-width*
	Width of the sidebar, including the border. Set to zero to disable the
	sidebar.

	Default: _20_

*empty-message*
	Message to display when viewing an empty folder.

	Default: _(no messages)_

*empty-dirlist*
	Message to display when no folders exists or are all filtered.

	Default: _(no folders)_

*mouse-enabled*
	Enable mouse events in the ui, e.g. clicking and scrolling with the mousewheel

	Default: _false_

*new-message-bell*
	Ring the bell when a new message is received.

	Default: _true_

*pinned-tab-marker*
	Marker to show before a pinned tab's name.

	Default: _`_

*spinner*
	Animation shown while loading, split by *spinner-delimiter* (below)

	Examples:
	- *spinner* = _"\-\_-,\_-\_"_
	- *spinner* = _'. , .'_
	- *spinner* = _"\,|,/,-"_

	Default: _"[..]    , [..]   ,  [..]  ,   [..] ,    [..],   [..] ,  [..]  , [..]   "_

*spinner-delimiter*
	Spinner delimiter to split string into an animation

	Default: _,_

*sort*
	List of space-separated criteria to sort the messages by, see *sort*
	command in *aerc*(1) for reference. Prefixing a criterion with _-r_
	reverses that criterion.

	Example:
		*sort* = _from -r date_

*dirlist-format*
	Describes the format string to use for the directory list.

	Default: _%n %>r_

[- *Format specifier*
:[ *Description*
|  _%%_
:  literal %
|  _%n_
:  directory name
|  _%N_
:  compacted directory name
|  _%r_
:  recent/unseen/total message count
|  _%>X_
:  make format specifier 'X' be right justified

*dirlist-delay*
	Delay after which the messages are actually listed when entering
	a directory. This avoids loading messages when skipping over folders
	and makes the UI more responsive. If you do not want that, set it to
	_0s_.

	Default: _200ms_

*dirlist-tree*
	Display the directory list as a foldable tree.

	Default: _false_

*dirlist-collapse*
	If *dirlist-tree* is enabled, set level at which folders are collapsed
	by default. Set to _0_ to disable.

	Default: _0_

*next-message-on-delete*
	Moves to next message when the current message is deleted, archived, or moved.

	Default: _true_

*auto-mark-read*
	Set the _seen_ flag when a message is opened in the message viewer.

	Default: _true_

*completion-popovers*
	Shows potential auto-completions for text inputs in popovers.

	Default: _true_

*completion-delay*
	How long to wait after the last input before auto-completion is triggered.

	Default: _250ms_

*completion-min-chars*
	The minimum required characters to allow auto-completion to be triggered
	after *completion-delay*.

	Default: _1_

*border-char-vertical*++
*border-char-horizontal*
	Set stylable characters (via the *border* element) for vertical and
	horizontal borders.

	Default: _" "_

*stylesets-dirs*
	The directories where the stylesets are stored. The config takes
	a colon-separated list of dirs. If this is unset or if a styleset cannot
	be found, the following paths will be used as a fallback in that order:

	```
	${XDG_CONFIG_HOME:-~/.config}/aerc/stylesets
	${XDG_DATA_HOME:-~/.local/share}/aerc/stylesets
	/usr/local/share/aerc/stylesets
	/usr/share/aerc/stylesets
	```

*styleset-name*
	The name of the styleset to be used to style the ui elements. The
	stylesets are stored in the _stylesets_ directory in the config
	directory.

	Default: _default_

	Have a look at *aerc-stylesets*(7) as to how a styleset looks like.

*icon-unencrypted*
	The icon to display for unencrypted mails. The status indicator is only
	displayed if an icon is set.

*icon-encrypted*
	The icon to display for encrypted mails.

	Default: _[e]_

*icon-signed*
	The icon to display for signed mails where the signature was
	successfully validated.

	Default: _[s]_

*icon-signed-encrypted*
	The icon to display for signed and encrypted mails where the signature
	was successfully verified. The combined icon is only used if set,
	otherwise the signed and encrypted icons are displayed separately.

*icon-unknown*
	The icon to display for signed mails which could not be verified due to
	the key being unknown.

	Default: _[s?]_

*icon-invalid*
	The icon to display for signed mails where verification failed.

	Default: _[s!]_

*fuzzy-complete*
	When typing a command or option, the popover will now show not only the
	items /starting/ with the string input by the user, but it will also show
	instances of items /containing/ the string, starting at any position and
	need not be consecutive characters in the command or option.

*reverse-msglist-order*
	Reverses the order of the message list. By default, the message list is
	ordered with the newest (highest UID) message on top. Reversing the
	order will put the oldest (lowest UID) message on top. This can be
	useful in cases where the backend does not support sorting.

	Default: _false_

*reverse-thread-order*
	Reverse display of the mesage threads. Default order is the the intial
	message is on the top with all the replies being displayed below. The
	reverse option will put the initial message at the bottom with the
	replies on top.

	Default: _false_

*sort-thread-siblings*
	Sort the thread siblings according to the sort criteria for the messages. If
	sort-thread-siblings is false, the thread siblings will be sorted based on
	the message UID. This option is only applicable for client-side threading
	with a backend that enables sorting.

	Default: _false_

*threading-enabled*
	Enable a threaded view of messages. If this is not supported by the
	backend (IMAP server or notmuch), threads will be built by the client.

	Default: _false_

*force-client-threads*
	Force threads to be built client-side. Backends that don't support threading
	will always build threads client side.

	Default: _false_


## CONTEXTUAL UI CONFIGURATION

The UI configuration can be specialized for accounts, specific mail
directories and message subjects. The specializations are added using
contextual config sections based on the context.

The contextual UI configuration is merged to the base UiConfig in the
following order:
*Base UIConfig > Account Context > Folder Context > Subject Context.*

*[ui:account=*_AccountName_*]*
	Adds account specific configuration with the account name.

*[ui:folder=*_FolderName_*]*
	Add folder specific configuration with the folder name.

*[ui:folder~*_Regex_*]*
	Add folder specific configuration for folders whose names match the regular
	expression.

*[ui:subject~*_Regex_*]*
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

## STATUSLINE

These options are configured in the *[statusline]* section of _aerc.conf_.

*render-format*
	Describes the format string for the statusline format.

	For a minimal statusline that only shows the current account and
	the connection information, use _[%a] %c_.

	To completely mute the statusline (except for push notifications), use
	_%m_ only.

	Default: _[%a] %S %>%T_

[- *Format specifier*
:[ *Description*
|  _%%_
:  literal %
|  _%a_
:  active account name
|  _%d_
:  active directory name
|  _%c_
:  connection state
|  _%p_
:  current path
|  _%m_
:  mute statusline and show only push notifications
|  _%S_
:  general status information (e.g. connection state, filter, search)
|  _%T_
:  general on/off information (e.g. passthrough, threading, sorting)
|  _%>_
:  does not print anything but all format specifier that follow will be right justified.

*separator*
	Specifies the separator between grouped statusline elements (e.g. for
	the _%S_ and _%T_ specifiers in *render-format*).

	Default: _" | "_

*display-mode*
	Defines the mode for displaying the status elements.
	Options: _text_, _icon_

	Default: _text_

## VIEWER

These options are configured in the *[viewer]* section of _aerc.conf_.

*pager*
	Specifies the pager to use when displaying emails. Note that some filters
	may add ANSI escape sequences to add color to rendered emails, so you may
	want to use a pager which supports ANSI.

	Default: _less -R_

*alternatives*
	If an email offers several versions (multipart), you can configure which
	mimetype to prefer. For example, this can be used to prefer plaintext over
	HTML emails.

	Default: _text/plain,text/html_

*header-layout*
	Defines the default headers to display when viewing a message. To display
	multiple headers in the same row, separate them with a pipe, e.g. _From|To_.
	Rows will be hidden if none of their specified headers are present in the
	message.

	Notmuch tags can be displayed by adding Labels.

	Authentication information from the Authentication-Results header can be
	displayed by adding _DKIM_, _SPF_ or _DMARC_. To show more information
	than just the authentication result, append a plus sign (*+*) to the header name
	(e.g. _DKIM+_).

	Default: _From|To,Cc|Bcc,Date,Subject_

*show-headers*
	Default setting to determine whether to show full headers or only parsed
	ones in message viewer.

	Default: _false_

*always-show-mime*
	Whether to always show the mimetype of an email, even when it is just a single part.

	Default: _false_

*parse-http-links*
	Parses and extracts http links when viewing a message. Links can then be
	accessed with the *open-link* command.

	Default: _true_

## COMPOSE

These options are configured in the *[compose]* section of _aerc.conf_.

*editor*
	Specifies the command to run the editor with. It will be shown in an
	embedded terminal, though it may also launch a graphical window if the
	environment supports it.

	Defaults to *$EDITOR*, or *vi*(1).

*header-layout*
	Defines the default headers to display when composing a message. To display
	multiple headers in the same row, separate them with a pipe, e.g. _To|From_.

	Default: _To|From,Subject_

*address-book-cmd*
	Specifies the command to be used to tab-complete email addresses. Any
	occurrence of _%s_ in the *address-book-cmd* will be replaced with anything
	the user has typed after the last comma.

	The command must output the completions to standard output, one completion
	per line. Each line must be tab-delimited, with an email address occurring as
	the first field. Only the email address field is required. The second field,
	if present, will be treated as the contact name. Additional fields are
	ignored.

	This parameter can also be set per account in _accounts.conf_.

	Example:
		*address-book-cmd* = _khard email --remove-first-line --parsable %s_

*file-picker-cmd*
	Specifies the command to be used to select attachments. Any occurence of
	_%s_ in the *file-picker-cmd* will be replaced with the argument _<arg>_
	to *:attach -m* _<arg>_.

	The command must output the selected files to standard output,
	one file per line.

	Example:
		*file-picker-cmd* = _fzf --multi --query=%s_

*reply-to-self*
	If set to _false_, do not mail yourself when replying (e.g., if replying
	to emails previously sent by yourself, address your replies to the
	original To and Cc).

	Default: _true_

*no-attachment-warning*
	Specifies a regular expression against which an email's body should be
	tested before sending an email with no attachment. If the regexp
	matches, aerc will warn you before sending the message. Leave empty to
	disable this feature.

	Uses Go's regexp syntax, documented at https://golang.org/s/re2syntax.
	The _(?im)_ flags are set by default (case-insensitive and multi-line).

	Example:
		*no-attachment-warning* = _^[^>]\*attach(ed|ment)_

## FILTERS

Filters allow you to pipe an email body through a shell command to render
certain emails differently, e.g. highlighting them with ANSI escape codes.
They are configured in the *[filters]* section of _aerc.conf_.

The first filter which matches the email's mimetype will be used, so order
them from most to least specific.

You can also match on non-mimetypes, by prefixing with the header to match
against (non-case-sensitive) and a comma, e.g. _subject,text_ will match a
subject which contains _text_. Use _header,~regex_ to match against a _regex_.

aerc ships with some default filters installed in the share directory (usually
_/usr/share/aerc/filters_). Note that these may have additional dependencies
that aerc does not have alone.

The filter commands are invoked with _sh -c command_. The following folders are
appended to the system *$PATH* to allow referencing filters from their name only.

```
${XDG_CONFIG_HOME:-~/.config}/aerc/filters
${XDG_DATA_HOME:-~/.local/share}/aerc/filters
$PREFIX/share/aerc/filters
/usr/share/aerc/filters
```

The following variables are defined in the filter command environment:

*AERC_MIME_TYPE*
	the part MIME type/subtype
*AERC_FORMAT*
	the part content type format= parameter (e.g. format=flowed)
*AERC_FILENAME*
	the attachment filename (if any)
*AERC_SUBJECT*
	the message Subject header value
*AERC_FROM*
	the message From header value

Note that said email body is converted into UTF-8 before being passed to
filters.

Example:

```
[filters]
from,thatguywhodoesnothardwraphismessages=fmt -w 72 | colorize
subject,~Git(hub|lab)=lolcat -f
text/plain=colorize
text/calendar=calendar | colorize
message/delivery-status=colorize
message/rfc822=colorize
text/html=html | colorize
text/*=bat -fP --file-name="$AERC_FILENAME"
application/x-sh=bat -fP -l sh
image/*=catimg -w $(tput cols) -
```

## OPENERS

Openers allow you to specify the command to use for the *:open* action on a
per-MIME-type basis. They are configured in the *[openers]* section of
_aerc.conf_.

_{}_ is expanded as the temporary filename to be opened. If it is not
encountered in the command, the temporary filename will be appened to the end
of the command. Environment variables are also expanded. Tilde is not expanded.

Example:

```
[openers]
text/html=surf -dfgms
text/plain=gvim {} +125
message/rfc822=thunderbird
```

## TRIGGERS

Triggers specify commands to execute when certain events occur.

They are configured in the *[triggers]* section of _aerc.conf_.

*new-email*
	Executed when a new email arrives in the selected folder.

	e.g. new-email=exec notify-send "New email from %n" "%s"

	Format specifiers from *index-format* are expanded with respect to the new
	message.

## TEMPLATES

Templates are used to populate the body of an email. The *:compose*, *:reply*
and *:forward* commands can be called with the *-T* flag with the name of the
template name.

aerc ships with some default templates installed in the share directory (usually
_/usr/share/aerc/templates_).

These options are configured in the *[templates]* section of _aerc.conf_.

*template-dirs*
	The directory where the templates are stored. The config takes
	a colon-separated list of dirs. If this is unset or if a template cannot
	be found, the following paths will be used as a fallback in that order:

	```
	${XDG_CONFIG_HOME:-~/.config}/aerc/templates
	${XDG_DATA_HOME:-~/.local/share}/aerc/templates
	/usr/local/share/aerc/templates
	/usr/share/aerc/templates
	```

*new-message*
	The default template to be used for new messages.

	Default: _new_message_

*quoted-reply*
	The default template to be used for quoted replies.

	Default: _quoted_reply_

*forwards*
	The default template to be used for forward as body.

	Default: _forward_as_body_

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

	Default: _Archive_

*check-mail*
	Specifies an interval to check for new mail. Mail will be checked at
	startup, and every interval. IMAP accounts will check for mail in all
	unselected folders, and the selected folder will continue to receive PUSH
	mail notifications. Maildir/Notmuch folders must use *check-mail-cmd* in
	conjunction with this option. See *aerc-maildir* and *aerc-notmuch* for
	more information.

	Setting this option to _0_ will disable *check-mail*

	Example:
		*check-mail* = _5m_

	Default: _0_

*copy-to*
	Specifies a folder to copy sent mails to, usually _Sent_.

*default*
	Specifies the default folder to open in the message list when aerc
	configures this account.

	Default: _INBOX_

*folders*
	Specifies the comma separated list of folders to display in the sidebar.
	Names prefixed with _~_ are interpreted as regular expressions.

	Default: all folders

*folders-exclude*
	Specifies the comma separated list of folders to exclude from the sidebar.
	Names prefixed with _~_ are interpreted as regular expressions.
	Note that this overrides anything from *folders*.

	Default: no folders

*enable-folders-sort*
	If _true_, folders are sorted, first by specified folders (see *folders-sort*),
	then alphabetically.

	Default: _true_

*folders-sort*
	Specifies a comma separated list of folders to be shown at the top of the
	list in the provided order. Remaining folders will be sorted alphabetically.

*from*
	The default value to use for the From header in new emails. This should be
	an RFC 5322-compatible string, such as _Your Name <you@example.org>_.

*aliases*
	All aliases of the current account. These will be used to fill in the From:
	field. Make sure that your email server accepts this value, or for example
	use *aerc-sendmail*(5) in combination with *msmtp*(1) and
	*--read-envelope-from*.

*outgoing*
	Specifies the transport for sending outgoing emails on this account. It
	should be a connection string, and the specific meaning of each component
	varies depending on the protocol in use. See each protocol's man page for
	more details:

	- *aerc-smtp*(5)

*outgoing-cred-cmd*
	Specifies an optional command that is run to get the outgoing account's
	password. See each protocol's man page for more details.

*outgoing-cred-cmd-cache*
	By default, the credentials returned by the command will be cached until
	aerc is shut down. If set to _false_, *outgoing-cred-cmd* will be executed
	every time an email is to be sent.

	Default: _true_

*pgp-auto-sign*
	If _true_, all outgoing emails from this account will be signed (if a signing
	key is available).

	Default: _false_

*pgp-key-id*
	Specify the key id to use when signing a message. Can be either short or
	long key id. If unset, aerc will look up the key by email.

*pgp-opportunistic-encrypt*
	If _true_, any outgoing email from this account will be encrypted when all
	recipients (including Cc and Bcc field) have a public key available in
	the keyring.

	Default: _false_

*postpone*
	Specifies the folder to save postponed messages to.

	Default: _Drafts_

*send-as-utc*
	Converts the timestamp of the Date header to UTC.

	Default: _false_

*source*
	Specifies the source for reading incoming emails on this account. This key
	is required for all accounts. It should be a connection string, and the
	specific meaning of each component varies depending on the protocol in use.
	See each protocol's man page for more details:

	- *aerc-imap*(5)
	- *aerc-maildir*(5)
	- *aerc-notmuch*(5)

*source-cred-cmd*
	Specifies an optional command that is run to get the source account's
	password. See each protocol's man page for more details.

*signature-file*
	Specifies the file to read in order to obtain the signature to be added
	to emails sent from this account.

*signature-cmd*
	Specifies the command to execute with _sh -c_ in order to obtain the
	signature to be added to emails sent from this account. If the command
	fails then *signature-file* is used instead.

*trusted-authres*
	Comma-separated list of trustworthy hostnames from which the
	Authentication Results header will be displayed. Entries can be regular
	expressions. If you want to trust any host (e.g. for debugging),
	use the wildcard _\*_.

*subject-re-pattern*
	When replying to a message, this is the regular expression that will
	be used to match the prefix of the original message's subject that has
	to be removed, to create the subject line of the new message.
	Typically, this will be used to avoid a repetition of the Re:
	prefix in the subject header. The default will match known
	translations for the common Re:.

	Default: _(?i)^((AW|RE|SV|VS|ODP|R): ?)+_

# BINDS.CONF

This file is used for configuring keybindings used in the aerc interactive
client. You may configure different keybindings for different contexts by
writing them into different *[sections]* of the ini file. The available contexts
are:

*[messages]*
	keybindings for the message list

*[view]*
	keybindings for the message viewer

*[view::passthrough]*
	keybindings for the viewer, when in key passthrough mode
	(toggled with *:toggle-key-passthrough*)

*[compose]*
	keybindings for the message composer

*[compose::editor]*
	keybindings for the composer, when the editor is focused

*[compose::review]*
	keybindings for the composer, when reviewing the email before it's sent

*[terminal]*
	keybindings for terminal tabs

You may also configure account specific key bindings for each context:

*[context:account=*_AccountName_*]*
	keybindings for this context and account, where _AccountName_ matches
	the account name you provided in _accounts.conf_.

Folder-specific bindings can be configured for message lists:

*[messages:folder=*_FolderName_*]*
	keybindings under this section will be specific to the folder named
	_FolderName_. Keybindings from a folder specifier will take precedence
	over account specifiers

Example:

```
[messages:account=Mailbox]
c = :cf path:mailbox/** and<space>

[compose::editor:account=Mailbox2]

[messages:folder=Drafts]
<Enter> = :recall<Enter>
...
```

You may also configure global keybindings by placing them at the beginning of
the file, before specifying any context-specific sections. For each *key=value*
option specified, the _key_ is the keystrokes pressed (in order) to invoke this
keybinding, and _value_ specifies keystrokes that aerc will simulate when the
keybinding is invoked. Generally this is used to execute commands, for example:

	rq = :reply -q<Enter>

Pressing *r*, then *q*, will simulate typing in *:reply -q<Enter>*, and execute
*:reply -q* accordingly. It is also possible to invoke keybindings recursively in
a similar fashion. Additionally, the following special options are available in
each binding context:

*$noinherit*
	If set to _true_, global keybindings will not be effective in this context.

	Default: _false_

*$ex*
	This can be set to a keystroke which will bring up the command input in this
	context.

	Default: _:_

In addition to letters, special keys may be specified in *<angle brackets>*. The
following special keys are supported:

[[ *Name*
:- *Description*
|  *<space>*
:  " "
|  *<semicolon>*
:  ;
|  *<tab>*
:  Tab
|  *<enter>*
:  Enter
|  *<up>*
:  Up arrow
|  *<c-up>*
:  Ctrl+Up
|  *<a-up>*
:  Alt+Up
|  *<down>*
:  Down arrow
|  *<c-down>*
:  Ctrl+Down
|  *<a-down>*
:  Alt+Down
|  *<right>*
:  Right arrow
|  *<c-right>*
:  Ctrl+Right
|  *<a-right>*
:  Alt+Right
|  *<left>*
:  Left arrow
|  *<c-left>*
:  Ctrl+Left
|  *<a-left>*
:  Alt+Left
|  *<pgup>*
:  Page Up
|  *<c-pgup>*
:  Ctrl+PageUp
|  *<a-pgup>*
:  Alt+PageUp
|  *<pgdn>*
:  Page Down
|  *<c-pgdn>*
:  Ctrl+PageDn
|  *<a-pgdn>*
:  Alt+PageDn
|  *<home>*
:  Home
|  *<end>*
:  End
|  *<insert>*
:  Insert
|  *<delete>*
:  Delete
|  *<exit>*
:  Exit
|  *<cancel>*
:  Cancel
|  *<print>*
:  Print screen
|  *<pause>*
:  Pause
|  *<backtab>*
:  Shift+Tab
|  *<c-space>*
:  Ctrl+Space
|  *<a-space>*
:  Alt+Space
|  *<c-a>*
:  Ctrl+a
|  a-a
:  Alt+a
|  *<c-b>*
:  Ctrl+b
|  *<a-b>*
:  Alt+b
|  *<c-c>*
:  Ctrl+c
|  *<a-c>*
:  Alt+c
|  *<c-d>*
:  Ctrl+d
|  *<a-d>*
:  Alt+d
|  *<c-e>*
:  Ctrl+e
|  *<a-e>*
:  Alt+e
|  *<c-f>*
:  Ctrl+f
|  *<a-f>*
:  Alt+f
|  *<c-g>*
:  Ctrl+g
|  *<a-g>*
:  Alt+g
|  *<c-h>*
:  Ctrl+h
|  *<a-h>*
:  Alt+h
|  *<c-i>*
:  Ctrl+i
|  *<a-i>*
:  Alt+i
|  *<c-j>*
:  Ctrl+j
|  *<a-j>*
:  Alt+j
|  *<c-k>*
:  Ctrl+k
|  *<a-k>*
:  Alt+k
|  *<c-l>*
:  Ctrl+l
|  *<a-l>*
:  Alt+l
|  *<c-m>*
:  Ctrl+m
|  *<a-m>*
:  Alt+m
|  *<c-n>*
:  Ctrl+n
|  *<a-n>*
:  Alt+n
|  *<c-o>*
:  Ctrl+o
|  *<a-o>*
:  Alt+o
|  *<c-p>*
:  Ctrl+p
|  *<a-p>*
:  Alt+p
|  *<c-q>*
:  Ctrl+q
|  *<a-q>*
:  Alt+q
|  *<c-r>*
:  Ctrl+r
|  *<a-r>*
:  Alt+r
|  *<c-s>*
:  Ctrl+s
|  *<a-s>*
:  Alt+s
|  *<c-t>*
:  Ctrl+t
|  *<a-t>*
:  Alt+t
|  *<c-u>*
:  Ctrl+u
|  *<a-u>*
:  Alt+u
|  *<c-v>*
:  Ctrl+v
|  *<a-v>*
:  Alt+v
|  *<c-w>*
:  Ctrl+w
|  *<a-w>*
:  Alt+w
|  *<c-x>*
:  Ctrl+x
|  *<a-x>*
:  Alt+x
|  *<c-y>*
:  Ctrl+y
|  *<a-y>*
:  Alt+y
|  *<c-z>*
:  Ctrl+z
|  *<a-z>*
:  Alt+z
|  *<c-]>*
:  Ctrl+]
|  *<a-]>*
:  Alt+]
|  *<c-[>*
:  Ctrl+[
|  *<a-[>*
:  Alt+[
|  *<c-^>*
:  Ctrl+^
|  *<a-^>*
:  Alt+^
|  *<c-\_>*
:  Ctrl+\_
|  *<a-\_>*
:  Alt+\_

# SEE ALSO

*aerc*(1) *aerc-imap*(5) *aerc-smtp*(5) *aerc-maildir*(5) *aerc-sendmail*(5)
*aerc-notmuch*(5) *aerc-stylesets*(7)

# AUTHORS

Originally created by Drew DeVault <sir@cmpwn.com> and maintained by Robin
Jarry <robin@jarry.cc> who is assisted by other open source contributors. For
more information about aerc development, see https://sr.ht/~rjarry/aerc/.
