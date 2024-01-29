AERC-CONFIG(5)

# NAME

aerc-config - configuration file format for *aerc*(1)

# SYNOPSIS

There are three aerc config files: _aerc.conf_, _binds.conf_, and
_accounts.conf_. The last one must be kept secret, as it may include your
account credentials. We look for these files in your XDG config home plus
_aerc_, which defaults to _~/.config/aerc_. Alternate files can be specified via
command line arguments, see *aerc*(1).

Examples of these config files are typically included with your installation of
aerc and are usually installed in _/usr/share/aerc_.

Each file uses the ini format, and consists of sections with keys and values.
A line beginning with _#_ is considered a comment and ignored, as are empty
lines. New sections begin with _[section-name]_ on a single line, and keys and
values are separated with _=_.

This manual page focuses on _aerc.conf_. _binds.conf_ is detailed in
*aerc-binds*(5) and _accounts.conf_ in *aerc-accounts*(5).

_aerc.conf_ is used for configuring the general appearance and behavior of aerc.

# GENERAL OPTIONS

These options are configured in the *[general]* section of _aerc.conf_.

*default-save-path* = _<path>_
	Used as a default path for save operations if no other path is specified.

*pgp-provider* = _auto_|_gpg_|_internal_
	If set to _gpg_, aerc will use system gpg binary and keystore for all
	crypto operations. If set to _internal_, the internal openpgp keyring
	will be used. If set to _auto_, the system gpg will be preferred unless
	the internal keyring already exists, in which case the latter will be
	used.

	Default: _auto_

*unsafe-accounts-conf* = _true_|_false_
	By default, the file permissions of _accounts.conf_ must be restrictive
	and only allow reading by the file owner (_0600_). Set this option to
	_true_ to ignore this permission check. Use this with care as it may
	expose your credentials.

	Default: _false_

*log-file* = _<path>_
	Output log messages to specified file. A path starting with _~/_ is
	expanded to the user home dir. When redirecting aerc's output to a file
	using _>_ shell redirection, this setting is ignored and log messages
	are printed to stdout.

*log-level* = _trace_|_debug_|_info_|_warn_|_error_
	Only log messages above the specified level to *log-file*. Supported
	levels are: _trace_, _debug_, _info_, _warn_ and _error_. When
	redirecting aerc's output to a file using _>_ shell redirection, this
	setting is ignored and the log level is forced to _trace_.

	Default: _info_

*disable-ipc* = _true_|_false_
	Disable the execution of commands over IPC.

	Default: _false_

*term* = _<TERM>_
	Set the $TERM environment variable used for the embedded terminal.

	Default: _xterm-256color_

*enable-osc8* = _true_|_false_
	Enable the embedded terminal to output OSC 8 (hyperlinks) escape
	sequences. Not all terminal emulators handle OSC 8 sequences properly
	and can produce confusing results, disable this setting if that occurs.

	Default: _false_

*default-menu-cmd* = _<cmd>_
	Default shell command to use for *:menu*. This will be executed with
	_sh -c_ and will run in an popover dialog.

	Any occurrence of _%f_ will be replaced by a temporary file path where
	the command is expected to write output lines to be consumed by *:menu*.
	Otherwise, the lines will be read from the command's standard output.

	Example:
		*default-menu-cmd* = _fzf_

# UI OPTIONS

These options are configured in the *[ui]* section of _aerc.conf_.

*index-columns* = _<column1,column2,column3...>_
	Describes the format for each row in a mailbox view. This is a comma
	separated list of column names with an optional align and width suffix.
	After the column name, one of the _<_ (left), _:_ (center) or _>_
	(right) alignment characters can be added (by default, left) followed by
	an optional width specifier. The width is either an integer representing
	a fixed number of characters, or a percentage between _1%_ and _99%_
	representing a fraction of the terminal width. It can also be one of the
	_\*_ (auto) or _=_ (fit) special width specifiers. Auto width columns
	will be equally attributed the remaining terminal width. Fit width
	columns take the width of their contents. If no width specifier is set,
	_\*_ is used by default.

	Default: _date<20,name<17,flags>4,subject<\*_

*column-separator* = _"<separator>"_
	String separator inserted between columns. When a column width specifier
	is an exact number of characters, the separator is added to it (i.e. the
	exact width will be fully available for that column contents).

	Default: _"  "_

*column-<name>* = _<go template>_
	Each name in *index-columns* must have a corresponding *column-<name>*
	setting. All *column-<name>* settings accept golang text/template
	syntax.

	By default, these columns are defined:

	```
	column-date = {{.DateAutoFormat .Date.Local}}
	column-name = {{index (.From | names) 0}}
	column-flags = {{.Flags | join ""}}
	column-subject = {{.ThreadPrefix}}{{if .ThreadFolded}}{{printf "{%d}" .ThreadCount}}{{end}}{{.Subject}}
	```

	See *aerc-templates*(7) for all available symbols and functions.

*timestamp-format* = _<timeformat>_
	See time.Time#Format at https://godoc.org/time#Time.Format

	Default: _2006-01-02 03:04 PM_ (ISO 8601 + 12 hour time)

*this-day-time-format* = _<timeformat>_
	Index-only time format for messages that were received/sent today.
	If this is not specified, *timestamp-format* is used instead.

*this-week-time-format* = _<timeformat>_
	Index-only time format for messages that were received/sent within the
	last 7 days. If this is not specified, *timestamp-format* is used
	instead.

*this-year-time-format* = _<timeformat>_
	Index-only time format for messages that were received/sent this year.
	If this is not specified, *timestamp-format* is used instead.

*message-view-timestamp-format* = _<timeformat>_
	If set, overrides *timestamp-format* for the message view.

*message-view-this-day-time-format* = _<timeformat>_
	If set, overrides *timestamp-format* in the message view for messages
	that were received/sent today.

*message-view-this-week-time-format* = _<timeformat>_
	If set, overrides *timestamp-format* in the message view for messages
	that were recieved/sent within the last 7 days.

*message-view-this-year-time-format* = _<timeformat>_
	If set, overrides *timestamp-format* in the message view for messages
	that were received/sent this year.

*sidebar-width* = _<int>_
	Width of the sidebar, including the border. Set to zero to disable the
	sidebar.

	Default: _20_

*empty-message* = _<string>_
	Message to display when viewing an empty folder.

	Default: _(no messages)_

*empty-dirlist* = _<string>_
	Message to display when no folders exists or are all filtered.

	Default: _(no folders)_

*empty-subject* = _<string>_
	Text to display in message list, when the subject is empty.

	Default: _(no subject)_

*mouse-enabled* = _true_|_false_
	Enable mouse events in the ui, e.g. clicking and scrolling with the mousewheel

	Default: _false_

*new-message-bell* = _true_|_false_
	Ring the bell when a new message is received.

	Default: _true_

*tab-title-account* = _<go_template>_
	The template to use for account tab titles. See *aerc-templates*(7) for
	available field names. To conditionally show the unread count next to
	the account name, set to:

		*tab-title-account* = {{.Account}} {{if .Unread}}({{.Unread}}){{end}}

	Default: _{{.Account}}_

*tab-title-composer* = _<go_template>_
	The template to use for composer tab titles. See *aerc-templates*(7) for
	available field names.

	Default: _{{.Subject}}_

*tab-title-viewer* = _<go_template>_
	The template to use for viewer tab titles. See *aerc-templates*(7) for
	available field names.

	Default: _{{if .Subject}}{{.Subject}}{{else}}(no subject){{end}}_

*pinned-tab-marker* = _"<string>"_
	Marker to show before a pinned tab's name.

	Default: _`_

*spinner* = _"<string>"_
	Animation shown while loading, split by *spinner-delimiter* (below)

	Examples:
	- *spinner* = _"\-\_-,\_-\_"_
	- *spinner* = _'. , .'_
	- *spinner* = _"\,|,/,-"_

	Default: _"[..]    , [..]   ,  [..]  ,   [..] ,    [..],   [..] ,  [..]  , [..]   "_

*spinner-delimiter* = _<string>_
	Spinner delimiter to split string into an animation

	Default: _,_

*spinner-interval* = _<duration>_
	The delay between each spinner frame

	Default: _200ms_

*sort* = _<criteria>_
	List of space-separated criteria to sort the messages by, see *:sort*
	command in *aerc*(1) for reference. Prefixing a criterion with _-r_
	reverses that criterion.

	Example:
		*sort* = _from -r date_

*dirlist-left* = _<go template>_
	Template for the left side of the directory list.
	See *aerc-templates*(7) for all available fields and functions.

	Default: _{{.Folder}}_

*dirlist-right* = _<go template>_
	Template for the right side of the directory list.
	See *aerc-templates*(7) for all available fields and functions.

	Default: _{{if .Unread}}{{humanReadable .Unread}}/{{end}}{{if .Exists}}{{humanReadable .Exists}}{{end}}_

*dirlist-delay* = _<duration>_
	Delay after which the messages are actually listed when entering
	a directory. This avoids loading messages when skipping over folders
	and makes the UI more responsive. If you do not want that, set it to
	_0s_.

	Default: _200ms_

*dirlist-tree* = _true_|_false_
	Display the directory list as a foldable tree.

	Default: _false_

*dirlist-collapse* = _<int>_
	If *dirlist-tree* is enabled, set level at which folders are collapsed
	by default. Set to _0_ to disable.

	Default: _0_

*next-message-on-delete* = _true_|_false_
	Moves to next message when the current message is deleted, archived, or moved.

	Default: _true_

*auto-mark-read* = _true_|_false_
	Set the _seen_ flag when a message is opened in the message viewer.

	Default: _true_

*completion-popovers* = _true_|_false_
	Shows potential auto-completions for text inputs in popovers.

	Default: _true_

*completion-delay* = _<duration>_
	How long to wait after the last input before auto-completion is triggered.

	Default: _250ms_

*completion-min-chars* = _<int>_
	The minimum required characters to allow auto-completion to be triggered
	after *completion-delay*.

	Setting this to _manual_ disables automatic completion, leaving only the
	manually triggered completion with the *$complete* key (see
	*aerc-binds*(5) for more details).

	Default: _1_

*border-char-vertical* = _"<char>"_++
*border-char-horizontal* = _"<char>"_
	Set stylable characters (via the *border* element) for vertical and
	horizontal borders.

	Default: _" "_

*stylesets-dirs* = _<path1:path2:path3...>_
	The directories where the stylesets are stored. The config takes
	a colon-separated list of dirs. If this is unset or if a styleset cannot
	be found, the following paths will be used as a fallback in that order:

	```
	${XDG_CONFIG_HOME:-~/.config}/aerc/stylesets
	${XDG_DATA_HOME:-~/.local/share}/aerc/stylesets
	/usr/local/share/aerc/stylesets
	/usr/share/aerc/stylesets
	```

*styleset-name* = _<string>_
	The name of the styleset to be used to style the ui elements. The
	stylesets are stored in the _stylesets_ directory in the config
	directory.

	Default: _default_

	Have a look at *aerc-stylesets*(7) as to how a styleset looks like.

*icon-unencrypted* = _<string>_
	The icon to display for unencrypted mails. The status indicator is only
	displayed if an icon is set.

*icon-encrypted* = _<string>_
	The icon to display for encrypted mails.

	Default: _[e]_

*icon-signed* = _<string>_
	The icon to display for signed mails where the signature was
	successfully validated.

	Default: _[s]_

*icon-signed-encrypted* = _<string>_
	The icon to display for signed and encrypted mails where the signature
	was successfully verified. The combined icon is only used if set,
	otherwise the signed and encrypted icons are displayed separately.

*icon-unknown* = _<string>_
	The icon to display for signed mails which could not be verified due to
	the key being unknown.

	Default: _[s?]_

*icon-invalid* = _<string>_
	The icon to display for signed mails where verification failed.

	Default: _[s!]_

*icon-attachment* = _<string>_
	The icon to display in *column-flags* when the message has an
	attachment.

	Default: _a_

*icon-new* = _<string>_
	The icon to display in *column-flags* when the message is unread and
	new.

	Default: _N_

*icon-old* = _<string>_
	The icon to display in *column-flags* when the message is unread and
	old.

	Default: _O_

*icon-replied* = _<string>_
	The icon to display in *column-flags* when the message has been replied
	to.

	Default: _r_

*icon-flagged* = _<string>_
	The icon to display in *column-flags* when the message is flagged.

	Default: _!_

*icon-marked* = _<string>_
	The icon to display in *column-flags* when the message is marked.

	Default: _\*_

*icon-draft* = _<string>_
	The icon to display in *column-flags* when the message is a draft.

	Default: _d_

*icon-deleted* = _<string>_
	The icon to display in *column-flags* when the message has been deleted.

	Default: _X_

*fuzzy-complete* = _true_|_false_
	When typing a command or option, the popover will now show not only the
	items /starting/ with the string input by the user, but it will also show
	instances of items /containing/ the string, starting at any position and
	need not be consecutive characters in the command or option.

*reverse-msglist-order* = _true_|_false_
	Reverses the order of the message list. By default, the message list is
	ordered with the newest (highest UID) message on top. Reversing the
	order will put the oldest (lowest UID) message on top. This can be
	useful in cases where the backend does not support sorting.

	Default: _false_

*reverse-thread-order* = _true_|_false_
	Reverse display of the message threads. By default, the thread root is
	displayed at the top of the tree with all replies below. The reverse
	option will put the thread root at the bottom with replies on top.

	Default: _false_

*sort-thread-siblings* = _true_|_false_
	Sort the thread siblings according to the sort criteria for the messages. If
	sort-thread-siblings is false, the thread siblings will be sorted based on
	the message UID. This option is only applicable for client-side threading
	with a backend that enables sorting.

	Default: _false_

*threading-enabled* = _true_|_false_
	Enable a threaded view of messages. If this is not supported by the
	backend (IMAP server or notmuch), threads will be built by the client.

	Default: _false_

*force-client-threads* = _true_|_false_
	Force threads to be built client-side. Backends that don't support threading
	will always build threads client side.

	Default: _false_

*client-threads-delay* = _<duration>_
	Delay of inactivity after which the client threads are rebuilt. Setting
	this to _0s_ may introduce a noticeable lag when scrolling through the
	message list.

	Default: _50ms_

*show-thread-context* = _true_|_false_
	Enable showing of thread context. Note: this is not supported by all
	backends.

	Default: _false_

*msglist-scroll-offset* = _<int>_
	Set the scroll offset in number of lines from the top and bottom
	of the message list.

	Default: _0_

## THREAD PREFIX CUSTOMIZATION

You can fully customize the thread arrows appearance, which is defined by the
following configurable prefix parts:

*thread-prefix-tip* = _<string>_
	Define the arrow head.

	Default: _">"_

*thread-prefix-indent* = _<string>_
	Define the arrow indentation.

	Default: _" "_

*thread-prefix-stem* = _<string>_
	Define the vertical extension of the arrow.

	Default: _"│"_

*thread-prefix-limb* = _<string>_
	Define the horizontal extension of the arrow.

	Default: _""_

*thread-prefix-folded* = _<string>_
	Define the folded thread indicator.

	Default: _""_

*thread-prefix-unfolded* = _<string>_
	Define the unfolded thread indicator.

	Default: _""_

*thread-prefix-first-child* = _<string>_
	Define the first child connector.

	Default: _""_

*thread-prefix-has-siblings* = _<string>_
	Define the connector used if the message has siblings.

	Default: _├─_

*thread-prefix-lone* = _<string>_
	Define the connector used if the message has no parents and no children.

	Default: _""_

*thread-prefix-orphan* = _<string>_
	Define the connector used if the message has no parents and has children.

	Default: _""_

*thread-prefix-last-sibling* = _<string>_
	Define the connector for the last sibling.

	Default: _└─_

*thread-prefix-last-sibling-reverse* = _<string>_
	Define the connector for the last sibling in reversed threads.

	Default: _┌─_

*thread-prefix-first-child-reverse* = _<string>_

	Define the arrow appearance by selecting the first child connector in
	reversed threads.

	Default: _""_

*thread-prefix-orphan-reverse* = _<string>_
	Customize the reversed threads arrow appearance by selecting the
	connector used if the message has no parents and has children.

	Default: _""_


Default settings (mutt-style):

	```
	[PATCH aerc v5] ui: allow thread arrow customisation
	├─>[aerc/patches] build success
	├─>Re: [PATCH aerc v5] ui: allow thread arrow customisation
	├─
	└─>
	  ├─>
	  │ ├─>
	  │ └─>
	  │   └─>
	  └─>
	```

More compact, rounded threads that are also fold-aware:

	```
	┌[PATCH aerc v5] ui: allow thread arrow customisation
	├─[aerc/patches] build success
	├─Re: [PATCH aerc v5] ui: allow thread arrow customisation
	├+
	╰┬
	 ├┬
	 │├─
	 │╰┬
	 │ ╰─
	 ╰─
	```

```
thread-prefix-tip = ""
thread-prefix-indent = ""
thread-prefix-stem = "│"
thread-prefix-limb = "─"
thread-prefix-folded = "+"
thread-prefix-unfolded = ""
thread-prefix-first-child = "┬"
thread-prefix-has-siblings = "├"
thread-prefix-orphan = "┌"
thread-prefix-lone = " "
thread-prefix-last-sibling = "╰"
```

## CONTEXTUAL UI CONFIGURATION

The UI configuration can be specialized for accounts, specific mail
directories and message subjects. The specializations are added using
contextual config sections based on the context.

The contextual UI configuration is merged to the base UiConfig in the
following order: *Base UIConfig > Account Context > Folder Context*.

*[ui:account=*_AccountName_*]*
	Adds account specific configuration with the account name.

*[ui:folder=*_FolderName_*]*
	Add folder specific configuration with the folder name.

*[ui:folder~*_Regex_*]*
	Add folder specific configuration for folders whose names match the regular
	expression.

Example:
```
[ui:account=Work]
sidebar-width=...

[ui:folder=Sent]
index-format=...

[ui:folder~Archive/\d+/.*]
index-format=...
```

# STATUSLINE

These options are configured in the *[statusline]* section of _aerc.conf_.

*status-columns* = _<column1,column2,column3...>_
	Describes the format for the statusline. This is a comma separated list
	of column names with an optional align and width suffix. See
	*[ui].index-columns* for more details.

	To completely mute the statusline (except for push notifications),
	explicitly set *status-columns* to an empty string:

		status-columns=

	Default: _left<\*,center>=,right>\*_

*column-separator* = _"<separator>"_
	String separator inserted between columns. See *[ui].column-separator*
	for more details.

	Default: _" "_

*column-<name>* = _<go template>_
	Each name in *status-columns* must have a corresponding *column-<name>*
	setting. All *column-<name>* settings accept golang text/template
	syntax.

	By default, these columns are defined:

	```
	column-left = [{{.Account}}] {{.StatusInfo}}
	column-center = {{.PendingKeys}}
	column-right = {{.TrayInfo}}
	```

	See *aerc-templates*(7) for all available symbols and functions.

*separator* = _"<string>"_
	Specifies the separator between grouped statusline elements (e.g. for
	the _{{.ContentInfo}}_, _{{.TrayInfo}}_ and _{{.StatusInfo}}_ in
	*column-<name>*).

	Default: _" | "_

*display-mode* = _text_|_icon_
	Defines the mode for displaying the status elements.

	Default: _text_

# VIEWER

These options are configured in the *[viewer]* section of _aerc.conf_.

*pager* = _<command>_
	Specifies the pager to use when displaying emails. Note that some filters
	may add ANSI escape sequences to add color to rendered emails, so you may
	want to use a pager which supports ANSI.

	Default: _less -Rc_

*alternatives* = _<mime,types>_
	If an email offers several versions (multipart), you can configure which
	mimetype to prefer. For example, this can be used to prefer plaintext over
	HTML emails.

	Default: _text/plain,text/html_

*header-layout* = _<header|layout,list...>_
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

*show-headers* = _true_|_false_
	Default setting to determine whether to show full headers or only parsed
	ones in message viewer.

	Default: _false_

*always-show-mime* = _true_|_false_
	Whether to always show the mimetype of an email, even when it is just a single part.

	Default: _false_

*max-mime-height* = _height_
	Define the maximum height of the mimetype switcher before a scrollbar is
	used. The height of the mimetype switcher is restricted to half of the display height.
	If the provided value for the height is zero, the number of parts will
	be used as the height of the type switcher.

	Default: 0

*parse-http-links* = _true_|_false_
	Parses and extracts http links when viewing a message. Links can then be
	accessed with the *open-link* command.

	Default: _true_

# COMPOSE

These options are configured in the *[compose]* section of _aerc.conf_.

*editor* = _<command>_
	Specifies the command to run the editor with. It will be shown in an
	embedded terminal, though it may also launch a graphical window if the
	environment supports it.

	The following variables are defined in the editor's environment:

	*AERC_ACCOUNT*
		the name of the current account
	*AERC_ADDRESS_BOOK_CMD*
		the _address-book-cmd_ specified for the current account in
		_accounts.conf_

	Defaults to *$EDITOR*, or *vi*(1).

*header-layout* = _<header|layout,list...>_
	Defines the default headers to display when composing a message. To display
	multiple headers in the same row, separate them with a pipe, e.g. _To|From_.

	Default: _To|From,Subject_

*edit-headers* = _true_|_false_
	Edit headers directly into the text editor instead of having separate UI
	text inputs.

	When this is set to _true_, the *:cc*, *:bcc* and *:header* commands do
	not work, editing email headers are left to the text editor.
	*address-book-cmd* is not supported and address completion is left to
	the editor itself. *header-layout* is ignored.

	Default: _false_

*address-book-cmd* = _<command>_
	Specifies the command to be used to tab-complete email addresses. Any
	occurrence of _%s_ in the *address-book-cmd* will be replaced with anything
	the user has typed after the last comma.

	The command must output the completions to standard output, one completion
	per line. Each line must be tab-delimited, with an email address occurring as
	the first field. Only the email address field is required. The second field,
	if present, will be treated as the contact name. Additional fields are
	ignored.

	This parameter can also be set per account in _accounts.conf_.

	Example with *carddav-query*(1):
		*address-book-cmd* = _carddav-query %s_

	Example with *khard*(1):
		*address-book-cmd* = _khard email --remove-first-line --parsable %s_

*file-picker-cmd* = _<command>_
	Specifies the command to be used to select attachments. Any occurrence of
	_%s_ in the *file-picker-cmd* will be replaced with the argument _<arg>_
	to *:attach -m* _<arg>_. Any occurence of _%f_ will be replaced by the
	location of a temporary file, from which aerc will read the selected files.

	If _%f_ is not present, the command must output the selected files to
	standard output, one file per line. If it is present, then aerc does not
	capture the standard output and instead reads the files from the temporary
	file which should have the same format.

	Examples:
		*file-picker-cmd* = _fzf --multi --query=%s_
		*file-picker-cmd* = _ranger --choose-files=%f_

*reply-to-self* = _true_|_false_
	If set to _false_, do not mail yourself when replying (e.g., if replying
	to emails previously sent by yourself, address your replies to the
	original To and Cc).

	Default: _true_

*empty-subject-warning* = _true_|_false_
	Warn before sending an email with an empty subject.

	Default: _false_

*no-attachment-warning* = _<regexp>_
	Specifies a regular expression against which an email's body should be
	tested before sending an email with no attachment. If the regexp
	matches, aerc will warn you before sending the message. Leave empty to
	disable this feature.

	Uses Go's regexp syntax, documented at https://golang.org/s/re2syntax.
	The _(?im)_ flags are set by default (case-insensitive and multi-line).

	Example:
		*no-attachment-warning* = _^[^>]\*attach(ed|ment)_

*format-flowed* = _true_|_false_
	When set, aerc will generate _Format=Flowed_ bodies with a content type
	of _"text/plain; Format=Flowed"_ as described in RFC3676. This format is
	easier to handle for some mailing software, and generally just looks
	like ordinary text. To actually make use of this format's features,
	you'll need support in your editor.

	Default: _false_

*lf-editor* = _true_|_false_
	By default, aerc will use RFC2822 standard _\\r\\n_ (CRLF) line breaks
	when composing messages. Use this option for text editors that only
	support non-standard _\\n_ (LF) line breaks.

	Default: _false_

# MULTIPART CONVERTERS

Converters allow generating _multipart/alternative_ messages by converting the
main _text/plain_ body into any other text MIME type with the *:multipart*
command. Only exact MIME types are accepted. The commands are invoked with
_sh -c_ and are expected to output valid UTF-8 text.

Only _text/<subtype>_ MIME parts can be generated. The _text/plain_ MIME type is
reserved and cannot be generated. You still need to write your emails by hand in
your favorite text editor.

Converters are configured in the *[multipart-converters]* section of
_aerc.conf_.

Example:

```
[multipart-converters]
text/html=pandoc -f markdown -t html --standalone
```

Obviously, this requires that you write your main _text/plain_ body using the
markdown syntax. Also, mind that some mailing lists reject emails that contain
_text/html_ alternative parts. Use this feature carefully and when possible,
avoid using it at all.

# FILTERS

Filters are a flexible and powerful way of handling viewing parts of an opened
message. When viewing messages aerc will show the list of available message
parts and their MIME type at the bottom, but unless a filter is defined for
a specific MIME type, it will only show a menu with a few options (allowing you
to open the part in an external program, save it to disk or pipe it to a shell
command). Configuring a filter will allow viewing the output of the filter in
the configured *pager* in aerc's built-in terminal.

Filters are configured in the *[filters]* section of *aerc.conf*. The first
filter which matches the part's MIME type will be used, so order them from most
to least specific. You can also match on non-MIME types, by prefixing with the
header to match against (non-case-sensitive) and a comma, e.g. _subject,text_
will match a subject which contains _text_. Use _header,~regex_ to match
against a _regex_.

Note that aerc will pipe the content into the configured filter program, so
filters need to be able to read from standard input. Many programs support
reading from stdin by putting _-_ instead of a path to a file. You can also
chain together multiple filters by piping with _|_.

aerc ships with some default filters installed in the libexec directory (usually
_/usr/libexec/aerc/filters_). Note that these may have additional dependencies
that aerc does not have alone.

The filter commands are invoked with _sh -c command_. The following folders are
prepended to the system *$PATH* to allow referencing filters from their name only.

```
${XDG_CONFIG_HOME:-~/.config}/aerc/filters
~/.local/libexec/aerc/filters
${XDG_DATA_HOME:-~/.local/share}/aerc/filters
$PREFIX/libexec/aerc/filters
$PREFIX/share/aerc/filters
/usr/libexec/aerc/filters
/usr/share/aerc/filters
```

If you want to run a program in your default *$PATH* which has the same
name as a builtin filter (e.g. _/usr/bin/colorize_), use its absolute path.

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

If *show-headers* is enabled, only the currently viewed part body is piped into
the filter command. A special _.headers_ filter command can be defined to post
process the full headers.

## EXAMPLES

_text/plain_
	Color some things, e.g. quotes, git diffs, links, etc.:

	```
	text/plain=colorize
	```

	The built-in _colorize_ filter can be configured in the *[viewer]*
	section of styleset files. See *aerc-stylesets*(7).

	Wrap long lines at 100 characters, while not messing up nested quotes.
	Handles format=flowed emails properly:

	```
	text/plain=wrap -w 100 | colorize
	```

_from,<sender>_
	Another example of hard wrapping lines of emails sent by a specific
	person. Explicitly reflow all paragraphs instead of only wrapping long
	lines. This may break manual formatting in some messages:

	```
	from,thatguywhoneverhardwrapshismessages=wrap -r -w 72 | colorize
	```

_subject,~<regexp>_
	Use rainbow coloring with *lolcat*(1) for emails sent by software
	forges:

	```
	subject,~Git(hub|lab)=lolcat -f
	```

_text/html_
	Render html to a more human readable version and colorize:

	```
	text/html=html | colorize
	```

	Use pandoc to output plain text:

	```
	text/html=pandoc -f html -t plain
	```

_text/calendar_
	Parse calendar invites:

	```
	text/calendar=calendar
	```

_text/\*_
	Catch any other type of text that did not have a specific filter and
	use *bat*(1) to color these:

	```
	text/\*=bat -fP --file-name="$AERC_FILENAME" --style=plain
	```

_.headers_
	Colorize email headers when *show-headers* is _true_.

	```
	.headers=colorize
	```

_message/delivery-status_
	When not being able to deliver the provider might send such emails:

	```
	message/delivery-status=colorize
	```

_message/rfc822_
	When getting emails as attachments, e.g. on some mailing lists digest
	format is sending an email with all the digest emails as attachments.
	Requires *caeml*(1) to be on *PATH*:

	```
	message/rfc822=caeml | colorize
	```

	https://github.com/ferdinandyb/caeml

_application/mbox_
	Emails as attachments in the mbox format. For example aerc can also
	create an mbox from messages with the *:pipe* command. Requires
	*catbox*(1) and *caeml*(1) to be on *PATH*:

	```
	application/mbox=catbox -c caeml | colorize
	```

	https://github.com/konimarti/catbox

_application/pdf_
	Render pdf to text and rewrap at 100 character width. Requires
	*pdftotext*(1) to be on *PATH*:

	```
	application/pdf=pdftotext - -l 10 -nopgbrk -q  - | fmt -w 100
	```

	https://www.xpdfreader.com/pdftotext-man.html

_image/\*_
	This is a tricky topic. It's possible to display images in a terminal,
	but for high resolution images the terminal you are using either needs
	to support sixels or the kitty terminal graphics protocol.
	Unfortunately, aerc's built-in terminal supports neither, so only highly
	pixelated images can be shown natively. A workaround is possible by
	asking the terminal to draw on top of aerc and then remove the image
	when done viewing.

	The built-in terminal can show pixelated images with *catimg*(1):

	```
	image/\*=catimg -w$(tput cols) -
	```

See the wiki at https://man.sr.ht/~rjarry/aerc/ for more examples and possible
customizations of the built-in filters.

# OPENERS

Openers allow you to specify the command to use for the *:open* and *:open-link*
actions on a per-MIME-type basis. The *:open-link* URL scheme is used to
determine the MIME type as follows: _x-scheme-handler/<scheme>_. They are
configured in the *[openers]* section of _aerc.conf_.

_{}_ is expanded as the temporary filename or URL to be opened with proper shell
quoting. If it is not encountered in the command, the filename/URL will be
appended to the end of the command. The command will then be executed with
_sh -c_.

Like *[filters]*, openers support basic shell globbing. The first opener which
matches the part's MIME type (or URL scheme handler MIME type) will be used, so
order them from most to least specific.

Example:

```
[openers]
x-scheme-handler/irc=hexchat
x-scheme-handler/http\*=printf '%s' {} | wl-copy
text/html=surf -dfgms
text/plain=gvim {} +125
message/rfc822=thunderbird
```

# HOOKS

Hooks are triggered whenever the associated event occurs. The commands are run
in a shell environment with information added to environment variables.

They are configured in the *[hooks]* section of aerc.conf.

*aerc-startup* = _<command>_
	Executed when aerc is started is received in the selected folder. If it
	is used to run certain commands at startup. The hook is executed as soon
	as the UI is initialized and does not wait for all accounts to be fully
	loaded.

	Variables:

	- *AERC_VERSION*
	- *AERC_BINARY*

	Example:

		*aerc-startup* = _aerc :terminal calcurse && aerc :next-tab_

*mail-received* = _<command>_
	Executed when new mail is received in the selected folder. This will
	only work reliably with maildir and some imap servers.

	Variables:

	- *AERC_ACCOUNT*
	- *AERC_FOLDER*
	- *AERC_FROM_NAME*
	- *AERC_FROM_ADDRESS*
	- *AERC_SUBJECT*

	Example:

		*mail-received* = _notify-send "[$AERC_ACCOUNT/$AERC_FOLDER] New mail from $AERC_FROM_NAME" "$AERC_SUBJECT"_

*mail-deleted* = _<command>_
	Executed when a message is deleted from a folder. Note that this hook is
	triggered when moving a message from one folder to another.

	Variables:

	- *AERC_ACCOUNT*
	- *AERC_FOLDER*

	Example:

		*mail-deleted* = _mbsync "$AERC_ACCOUNT:$AERC_FOLDER"_

*mail-added* = _<command>_
	Executed when a message is added to a folder. Note that this hook is not
	triggered when a new message is received (use *mail-received* for that) but
	rather is only triggered when aerc itself adds a message to a folder, e.g.
	when moving or copying a message.

	Variables:

	- *AERC_ACCOUNT*
	- *AERC_FOLDER*

	Example:

		*mail-added* = _mbsync "$AERC_ACCOUNT:$AERC_FOLDER"_

*mail-sent* = _<command>_
	Executed when a message is sent. This does not necessarily signify
	successful posting, if a queueing system like msmtpq is used.

	Variables:

	- *AERC_ACCOUNT*
	- *AERC_FROM_NAME*
	- *AERC_FROM_ADDRESS*
	- *AERC_SUBJECT*
	- *AERC_TO*
	- *AERC_CC*

	Example:

		*mail-sent* = _if [ "$AERC_ACCOUNT" = "gmail" ]; then mbsync
		gmail; fi_

*aerc-shutdown* = _<command>_
	Executed when aerc shuts down. Aerc will wait for the command to finish
	before exiting.

	Variables:

	- *AERC_LIFETIME*

# TEMPLATES

Template files are used to populate the body of an email. The *:compose*,
*:reply* and *:forward* commands can be called with the *-T* flag with the name
of the template name. The available symbols and functions are described in
*aerc-templates*(7).

aerc ships with some default templates installed in the share directory (usually
_/usr/share/aerc/templates_).

These options are configured in the *[templates]* section of _aerc.conf_.

*template-dirs* = _<path1:path2:path3...>_
	The directory where the templates are stored. The config takes
	a colon-separated list of dirs. If this is unset or if a template cannot
	be found, the following paths will be used as a fallback in that order:

	```
	${XDG_CONFIG_HOME:-~/.config}/aerc/templates
	${XDG_DATA_HOME:-~/.local/share}/aerc/templates
	/usr/local/share/aerc/templates
	/usr/share/aerc/templates
	```

*new-message* = _<template_name>_
	The default template to be used for new messages.

	Default: _new_message_

*quoted-reply* = _<template_name>_
	The default template to be used for quoted replies.

	Default: _quoted_reply_

*forwards* = _<template_name>_
	The default template to be used for forward as body.

	Default: _forward_as_body_

# SEE ALSO

*aerc*(1) *aerc-accounts*(5) *aerc-binds*(5) *aerc-imap*(5) *aerc-jmap*(5)
*aerc-maildir*(5) *aerc-notmuch*(5) *aerc-templates*(7) *aerc-sendmail*(5)
*aerc-smtp*(5) *aerc-stylesets*(7) *carddav-query*(1)

# AUTHORS

Originally created by Drew DeVault and maintained by Robin Jarry who is assisted
by other open source contributors. For more information about aerc development,
see _https://sr.ht/~rjarry/aerc/_.
