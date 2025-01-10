AERC(1)

# NAME

aerc - a pretty good email client.

# SYNOPSIS

*aerc* [*-h*] [*-v*] [*-a* _<name>_] [*-C* _<file>_] [*-A* _<file>_] [*-B*
_<file>_] [*-I*] [*mailto:*_<...>_ | *mbox:*_<file>_ | :_<command...>_]

For a guided tutorial, use *:help tutorial* from aerc, or *man aerc-tutorial*
from your terminal.

# OPTIONS

*-h*, *--help*
	Show aerc usage help and exit.

*-v*, *--version*
	Print the installed version of aerc and exit.

*-a* _<name>_++
*--account* _<name>_
	Load only the named account, as opposed to all configured accounts. It
	can also be a comma separated list of names. This option may be
	specified multiple times. The account order will be preserved.

*-C* _</path/to/aerc.conf>_++
*--aerc-conf* _</path/to/aerc.conf>_
	Instead of using _$XDG_CONFIG_HOME/aerc/aerc.conf_ use the file at the
	specified path for configuring aerc.

*-A* _</path/to/accounts.conf>_++
*--accounts-conf* _</path/to/accounts.conf>_
	Instead of using _$XDG_CONFIG_HOME/aerc/accounts.conf_ use the file at the
	specified path for configuring accounts.

*-B* _</path/to/binds.conf>_++
*--binds-conf* _</path/to/binds.conf>_
	Instead of using _$XDG_CONFIG_HOME/aerc/binds.conf_ use the file at the
	specified path for configuring binds.

*-I*, *--no-ipc*
	Run commands (*mailto:*_..._, *:*_<command...>_, *mbox:*_<file>_) directly
	in this instance rather than over IPC in an existing aerc instance. Also
	disable creation of an IPC server for subsequent aerc instances to
	communicate with this one.

*mailto:*_address[,address][?query[&query]]_
	Open the composer with the address(es) in the To field. These
	addresses must not be percent encoded.

	If aerc is already running (and IPC is not disabled), the composer is
	started in that instance; otherwise a new instance is started with the
	composer.

	The following (optional) query parameters are supported:

[[ *Query*
:[ *Description*
|  _subject=<text>_
:  Subject line will be completed with the _<text>_
|  _body=<text>_
:  Message body will be completed with the _<text>_
|  _cc=<address>[,<address>]_
:  Cc header will be completed with the list of addresses
|  _bcc=<address>[,<address>]_
:  Bcc header will be completed with the list of addresses
|  _in-reply-to=<message-id>_
:  In-reply-to header will be set to the message id
|  _account=<accountname>_
:  Specify the account (must be in _accounts.conf_; default is the selected account)
|  _template=<template-file>_
:  Template sets the template file for creating the message

	Note that reserved characters in the queries must be percent encoded.

*:*_<command...>_
	Run an aerc-internal command as you would in Ex-Mode. See *RUNTIME
	COMMANDS* below.

	The command to be executed and its arguments can either be passed as
	separate arguments in the shell (e.g., _aerc :cmd arg1 arg2_) or as a single
	argument in the shell (e.g., _aerc ":cmd arg1 arg2"_). In the former case,
	aerc may add quotes to the command before it is parsed in an attempt to
	preserve arguments containing spaces and other special characters. In the
	latter case, aerc will parse the command verbatim, as if it had been typed
	directly on aerc's command line. This latter form can be helpful for
	commands that don't interpret quotes in their arguments.

	If aerc is already running (and IPC is not disabled), the command is run in
	that instance; otherwise a new instance is started with the command.

*mbox:*_<file>_
	Open the specified mbox file as a virtual temporary account.

	If aerc is already running (and IPC is not disabled), the file is opened in
	that instance; otherwise a new instance is started with the file.

# RUNTIME COMMANDS

To execute a command, press *:* to bring up the command interface. Commands may
also be bound to keys, see *aerc-binds*(5) for details. In some contexts, such
as the terminal emulator, *<c-x>* is used to bring up the command interface.

Different commands work in different contexts, depending on the kind of tab you
have selected.

Dynamic arguments are expanded following *aerc-templates*(7) depending on the
context. For example, if you have a message selected, the following command:

```
:filter -f "{{index (.From | emails) 0}}"
```

Will filter all messages sent by the same sender.

Aerc stores a history of commands, which can be cycled through in command mode.
Pressing the up key cycles backwards in history, while pressing down cycles
forwards.

## GLOBAL COMMANDS

These commands work in any context.

*:help* _<topic>_++
*:man* _<topic>_
	Display one of aerc's man pages in the embedded terminal.

*:help* *keys*++
*:man* *keys*
	Display the active key bindings in the current context.

*:new-account* [*-t*]
	Start the new account wizard.

	*-t*: Create a temporary account. Do not modify _accounts.conf_.

*:cd* _<directory>_
	Changes aerc's current working directory.

*:z* _<directory or zoxide query>_
	Changes aerc's current working directory using zoxide. If zoxide is not on
	*$PATH*., the command will not be registered.

*:change-tab* [*+*|*-*]_<tab name or index>_++
*:ct* [*+*|*-*]_<tab name or index>_
	Changes the focus to the tab with the given name. If a number is given,
	it's treated as an index. If the number is prepended with *+* or *-*, the number
	is interpreted as a delta from the selected tab. If only a *-* is given, changes
	the focus to the previously selected tab.

*:exec* _<command>_
	Executes an arbitrary command in the background. Aerc will set the
	environment variables *$account* and *$folder* when the command is
	executed from an Account tab or an opened message.

	Note: commands executed in this way are not executed with the shell.

*:echo* _<string>_
	Resolve templates in _<string>_ and print it.

*:eml* [_<path>_]++
*:preview* [_<path>_]
	Opens an eml file and displays the message in the message viewer.

	Can also be used in the message viewer to open an rfc822 attachment or
	in the composer to preview the message.

*:pwd*
	Displays aerc's current working directory in the status bar.

*:send-keys* _<keystrokes>_
	Send keystrokes to the currently visible terminal, if any. Can be used to
	control embedded editors to save drafts or quit in a safe manner.

	Here's an example of quiting a Vim-like editor:

		*:send-keys* _<Esc>:wq!<Enter>_

	Note: when used in _binds.conf_ (see *aerc-binds*(5)), angle brackets
	need to be escaped in order to make their way to the command:

		<C-q> = :send-keys \\<Esc\\>:wq!\\<Enter\\><Enter>

	This way the _<Esc>_ and the first _<Enter>_ keystrokes are passed to
	*:send-keys*, while the last _<Enter>_ keystroke is executed directly,
	committing the *:send-keys* command's execution.

*:term* [_<command>..._]++
*:terminal* [_<command>..._]
	Opens a new terminal tab with a shell running in the current working
	directory, or the specified command.

*:move-tab* [_+_|_-_]_<index>_
	Moves the selected tab to the given index. If _+_ or _-_ is specified, the
	number is interpreted as a delta from the selected tab.

*:prev-tab* [_<n>_]++
*:next-tab* [_<n>_]
	Cycles to the previous or next tab in the list, repeating _<n>_ times
	(default: _1_).

*:pin-tab*
	Moves the current tab to the left of all non-pinned tabs and displays
	the *pinned-tab-marker* (default: _`_) to the left of the tab title.

*:unpin-tab*
	Removes the *pinned-tab-marker* from the current tab and returns the tab
	to its previous location.

*:prompt* _<prompt>_ _<command>..._
	Displays the prompt on the status bar, waits for user input, then appends
	that input as the last argument to the command and executes it. The input is
	passed as one argument to the command, unless it is empty, in which case no
	extra argument is added.

*:menu* [*-c* _"<shell-cmd>"_] [*-e*] [*-b*] [*-a*] [*-d*] _<aerc-cmd ...>_
	Opens a popover dialog running _sh -c "<shell-cmd>"_ (if not specified
	*[general].default-menu-cmd* will be used). When the command exits, all
	lines printed on its standard output will be appended to _<aerc-cmd ...>_
	and executed as a standard aerc command like *xargs*(1) would do when
	used in a shell. A colon (*:*) prefix is supported for _<aerc-cmd ...>_
	but is not required.

	*:menu* can be used without an external program by setting _<shell-cmd>_
	to _-_. This also acts as a fallback in case where no _<shell-cmd>_ was
	specified at all or the executable in the _<shell-cmd>_ was not found.

	*-c* _"<shell-cmd>"_
		Override *[general].default-menu-cmd*. See *aerc-config*(5) for
		more details.

	*-e*: Stop executing commands on the first error.

	*-b*: Do *NOT* spawn the popover dialog. Start the commands in the
	background (*NOT* in a virtual terminal). Use this if _<shell-cmd>_ is
	a graphical application that does not need a terminal.

	_<shell-cmd>_ may be fed with input text using the following flags:
		*-a*: All account names, one per line. E.g.:

			'<account>' LF

		*-d*: All current account directory names, one per line. E.g.:

			'<directory>' LF

		*-ad*: All directories of all accounts, one per line. E.g.:

			'<account>' '<directory>' LF

		Quotes may be added by aerc when either tokens contain special
		characters. The quotes should be preserved for _<aerc-cmd ...>_.

	Examples:

	```
	:menu -adc fzf :cf -a
	:menu -c 'fzf --multi' :attach
	:menu -dc 'fzf --multi' :cp
	:menu -bc 'dmenu -l 20' :cf
	:menu -c 'ranger --choosefiles=%f' :attach
	```

	This may also be used in key bindings (see *aerc-binds*(5)):

	```
	<C-p> = :menu -adc fzf :cf -a<Enter>
	```

*:choose* *-o* _<key>_ _<text>_ _<command>_ [*-o* _<key>_ _<text>_ _<command>_]...
	Prompts the user to choose from various options.

*:reload* [*-B*] [*-C*] [*-s* _<styleset-name>_]
	Hot-reloads the config files for the key binds and general *aerc* config.
	Reloading of the account config file is not supported.

	If no flags are provided, _binds.conf_, _aerc.conf_, and the current
	styleset will all be reloaded.

	*-B*: Reload _binds.conf_.

	*-C*: Reload _aerc.conf_.

	*-s* _<styleset-name>_
		Load the specified styleset.

*:suspend*
	Suspends the aerc process. Some ongoing connections may be terminated.

*:quit* [*-f*]++
*:exit* [*-f*]++
*:q*    [*-f*]
	Exits aerc. If a task is being performed that should not be interrupted
	(like sending a message), a normal quit call might fail. In this case,
	closing aerc can be forced with the *-f* option.

*:redraw*
	Force a full redraw of the screen.

## MESSAGE COMMANDS

These commands are valid in any context that has a selected message (e.g. the
message list, the message in the message viewer, etc).

*:archive* [*-m* _<strategy>_] _<scheme>_
	Moves the selected message to the archive. The available schemes are:

	_flat_: No special structure, all messages in the archive directory

	_year_: Messages are stored in folders per year

	_month_: Messages are stored in folders per year and subfolders per month

	The *-m* option sets the multi-file strategy. See *aerc-notmuch*(5) for more
	details.

*:accept* [*-e*|*-E*|*-s*]
	Accepts an iCalendar meeting invitation. This opens a compose window
	with a specially crafted attachment. Sending the email will let the
	inviter know that you accepted and will likely update their calendar as
	well. This will NOT add the meeting to your own calendar, that must be
	done as a separate manual step (e.g. by piping the text/calendar part to
	an appropriate script).

	*-e*: Forces *[compose].edit-headers* = _true_ for this message only.

	*-E*: Forces *[compose].edit-headers* = _false_ for this message only.

	*-s*: Skips the editor and goes directly to the review screen.

*:accept-tentative* [*-e*|*-E*]
	Accepts an iCalendar meeting invitation tentatively.

	*-e*: Forces *[compose].edit-headers* = _true_ for this message only.

	*-E*: Forces *[compose].edit-headers* = _false_ for this message only.

	*-s*: Skips the editor and goes directly to the review screen.

*:copy* [*-dp*] [*-a* _<account>_] [*-m* _<strategy>_] _<folder>_++
*:cp* [*-dp*] [*-a* _<account>_] [*-m* _<strategy>_] _<folder>_
	Copies the selected message(s) to _<folder>_.

	*-d*: Decrypt the message before copying.

	*-p*: Create _<folder>_ if it does not exist.

	*-a*: Copy to _<folder>_ of _<account>_. If _<folder>_ does
	not exist, it will be created whether or not *-p* is used.

	*-m*: Set the multi-file strategy. See *aerc-notmuch*(5) for more details.

*:decline* [*-e*|*-E*]
	Declines an iCalendar meeting invitation.

	*-e*: Forces *[compose].edit-headers* = _true_ for this message only.

	*-E*: Forces *[compose].edit-headers* = _false_ for this message only.

	*-s*: Skips the editor and goes directly to the review screen.

*:delete* [*-m* _<strategy>_]++
*:delete-message* [*-m* _<strategy>_]
	Deletes the selected message.

	*-m*: Set the multi-file strategy. See *aerc-notmuch*(5) for more details.

*:envelope* [*-h*] [*-s* _<format-specifier>_]
	Opens the message envelope in a dialog popup.

	*-h*: Show all header fields

	*-s* _<format-specifier>_
		User-defined format specifier requiring two _%s_ for the key and
		value strings. Default format: _%-20.20s: %s_

*:recall* [*-f*] [*-e*|*-E*]
	Opens the selected message for re-editing. Messages can only be
	recalled from the postpone directory.

	*-f*: Open the message for re-editing even if it is not in the postpone
	directory. Aerc remembers the folder, so the further *:postpone* call will
	save the message back there.

	*-e*: Forces *[compose].edit-headers* = _true_ for this message only.

	*-E*: Forces *[compose].edit-headers* = _false_ for this message only.

	Original recalled messages are deleted if they are sent or postponed again.
	In both cases you have another copy of the message somewhere. Otherwise the
	recalled message is left intact. This happens if the recalled message is
	discarded after editing. It can be deleted with *:rm* if it is not needed.

*:forward* [*-A*|*-F*] [*-T* _<template-file>_] [*-e*|*-E*] [_<address>_...]
	Opens the composer to forward the selected message to another recipient.

	*-A*: Forward the message and all attachments.

	*-F*: Forward the full message as an RFC 2822 attachment.

	*-T* _<template-file>_
		Use the specified template file for creating the initial
		message body. Unless *-F* is specified, this defaults to what
		is set as *forwards* in the *[templates]* section of
		_aerc.conf_.

	*-e*: Forces *[compose].edit-headers* = _true_ for this message only.

	*-E*: Forces *[compose].edit-headers* = _false_ for this message only.

*:move* [*-p*] [*-a* _<account>_] [*-m* _<strategy>_] _<folder>_++
*:mv* [*-p*] [*-a* _<account>_] [*-m* _<strategy>_] _<folder>_
	Moves the selected message(s) to _<folder>_.

	*-p*: Create _<folder>_ if it does not exist.

	*-a*: Move to _<folder>_ of _<account>_. If _<folder>_ does
	not exist, it will be created whether or not *-p* is used.

	*-m*: Set the multi-file strategy. See *aerc-notmuch*(5) for more details.

*:patch* _<args ...>_
	Patch management sub-commands. See *aerc-patch*(7) for more details.

*:pipe* [*-bdmps*] _<cmd>_
	Downloads and pipes the selected message into the given shell command
	(executed with _sh -c "<cmd>"_), and opens a new terminal tab to show
	the result. By default, the selected message part is used in the message
	viewer and the full message is used in the message list.

	Operates on multiple messages when they are marked. When piping multiple
	messages, aerc will write them with mbox format separators.

	*-b*: Run the command in the background instead of opening a terminal tab

	*-d*: Pipe the (full) message but decrypt it first.

	*-m*: Pipe the full message

	*-p*: Pipe just the selected message part, if applicable

	*-s*: Silently close the terminal tab after the command is completed

	This can be used to apply patch series with git:

		*:pipe -m* _git am -3_

	When at least one marked message subject matches a patch series (e.g.
	_[PATCH X/Y]_), all marked messages will be sorted by subject to ensure
	that the patches are applied in order.

*:reply* [*-acfq*] [*-T* _<template-file>_] [*-A* _<account>_] [*-e*|*-E*]
	Opens the composer to reply to the selected message.

	*-a*: Reply all

	*-c*: Close the view tab when replying. If the reply is not sent, reopen
	the view tab.

	*-f:* Reply to all addresses in From and Reply-To headers.

	*-q*: Insert a quoted version of the selected message into the reply
	editor. This defaults to what is set as *quoted-reply* in the *[templates]*
	section of _aerc.conf_.

	*-T* _<template-file>_
		Use the specified template file for creating the initial
		message body.

	*-A* _<account>_
		Reply with the specified account instead of the current one.

	*-e*: Forces *[compose].edit-headers* = _true_ for this message only.

	*-E*: Forces *[compose].edit-headers* = _false_ for this message only.

*:read* [*-t*]
	Marks the marked or selected messages as read.

	*-t*: Toggle the messages between read and unread.

*:unread* [*-t*]
	Marks the marked or selected messages as unread.

	*-t*: Toggle the messages between read and unread.

*:flag* [*-t*] [*-a* | *-x* _<flag>_]
	Sets (enables) a certain flag on the marked or selected messages.

	*-t*: Toggle the flag instead of setting (enabling) it.

	*-a*: Mark message as answered/unanswered.

	*-x* _<flag>_: Mark message with specific flag.
		The available flags are (adapted from RFC 3501, section 2.3.2):

		_Seen_
			Message has been read
		_Answered_
			Message has been answered
		_Forwarded_
			Message has been forwarded
		_Flagged_
			Message is flagged for urgent/special attention
		_Draft_
			Message is a draft

*:unflag* [*-t*] _<flag>_
	Operates exactly like *:flag*, defaulting to unsetting (disabling) flags.

*:modify-labels* [_+_|_-_]_<label>_...++
*:tag* [_+_|_-_]_<label>_...
	Modify message labels (e.g. notmuch tags). Labels prefixed with a *+* are
	added, those prefixed with a *-* removed. As a convenience, labels without
	either operand add the specified label.

	Example: add _inbox_ and _unread_ labels, remove _spam_ label.

		*:modify-labels* _+inbox_ _-spam_ _unread_

*:unsubscribe* [*-e*|*-E*]
	Attempt to automatically unsubscribe the user from the mailing list through
	use of the List-Unsubscribe header. If supported, aerc may open a compose
	window pre-filled with the unsubscribe information or open the unsubscribe
	URL in a web browser.

	*-e*: Forces *[compose].edit-headers* = _true_ for this message only.

	*-E*: Forces *[compose].edit-headers* = _false_ for this message only.

## MESSAGE LIST COMMANDS

*:align* _top|center|bottom_
	Aligns the selected message. The available positions are:

	_top_: Top of the message list.++
	_center_: Center of the message list.++
	_bottom_: Bottom of the message list.

*:disconnect*++
*:connect*
	Disconnect or reconnect the current account. This only applies to
	certain email sources.

*:clear* [*-s*]
	Clears the current search or filter criteria.

	By default, the selected message will be kept. To clear the selected
	message and move cursor to the top of the message list, use the *-s* flag.

	*-s*: Selects the message at the top of the message list after clearing.

*:cf* [*-a* _<account>_] _<folder>_
	Change the folder shown in the message list to _<folder>_.

	*-a* _<account>_
		Change to _<folder>_ of _<account>_ and focus its corresponding
		tab.

*:check-mail*
	Check for new mail on the selected account. Non-imap backends require
	check-mail-cmd to be set in order for aerc to initiate a check for new mail.
	Issuing a manual *:check-mail* command will reset the timer for automatic checking.

*:compose* [*-H* _"<header>: <value>"_] [*-T* _<template-file>_] [*-e*|*-E*] [_<body>_]
	Open the compose window to send a new email. The new email will be sent with
	the current account's outgoing transport configuration. For details on
	configuring outgoing mail delivery consult *aerc-accounts*(5).

	*-H* _"<header>: <value>"_
		Add the specified header to the message, e.g:

			*:compose -H* _"X-Custom: custom value"_

	*-T* _<template-file>_
		Use the specified template file for creating the initial
		message body.

	*-e*: Forces *[compose].edit-headers* = _true_ for this message only.

	*-E*: Forces *[compose].edit-headers* = _false_ for this message only.

	_<body>_: The initial message body.

*:bounce* [*-A* _<account>_] _<address>_ [_<address>_...]++
*:resend* [*-A* _<account>_] _<address>_ [_<address>_...]
	Bounce the selected message or all marked messages to the specified addresses,
	optionally using the specified account. This forwards the message while
	preserving all the existing headers. The new sender (*From*), date (*Date*),
	*Message-ID* and recipients (*To*) are prepended to the headers with the *Resent-*
	prefix. For more information please refer to section 3.6.6 of RFC 2822. Note
	that the bounced message is not copied over to the *sent* folder.

	Also please note that some providers (notably for instance Microsoft's
	O365) do not allow sending messages with the *From* header not matching
	any of the account's identities (even if *Resent-From* matches some).

*:recover* [*-f*] [*-e*|*-E*] _<file>_
	Resume composing a message that was not sent nor postponed. The file may
	not contain header data unless *[compose].edit-headers* was enabled when
	originally composing the aborted message.

	*-f*: Delete the _<file>_ after opening the composer.

	*-e*: Forces *[compose].edit-headers* = _true_ for this message only.

	*-E*: Forces *[compose].edit-headers* = _false_ for this message only.

*:filter* [_<options>_] _<terms>_...
	Similar to *:search*, but filters the displayed messages to only the search
	results. The search syntax is dependent on the underlying backend.
	Refer to *aerc-search*(1) for details

*:mkdir* _<name>_
	Creates a new folder for this account and changes to that folder.

*:rmdir* [*-f*] [_<folder>_]
	Removes the folder _<folder>_, or the current folder if not specified.

	By default, it will fail if the directory is non-empty (see *-f*).

	*-f*
		Remove the directory even if it contains messages.

	Some programs that sync maildirs may recover deleted directories (e.g.
	*offlineimap*). These can either be specially configured to properly
	handle directory deletion, or special commands need to be run to delete
	directories (e.g. _offlineimap --delete-folder_).

	It is possible, with a slow connection and the imap backend, that new
	messages arrive in the directory before they show up - using *:rmdir* at
	this moment would delete the directory and such new messages before the
	user sees them.

*:next* _<n>_[_%_]++
*:next-message* _<n>_[_%_]++
*:prev* _<n>_[_%_]++
*:prev-message* _<n>_[_%_]
	Selects the next (or previous) message in the message list. If specified as
	a percentage, the percentage is applied to the number of messages shown on
	screen and the cursor advances that far.

*:next-folder* _<n>_++
*:prev-folder* _<n>_
	Cycles to the next (or previous) folder shown in the sidebar, repeated
	_<n>_ times (default: _1_).

*:expand-folder* [_<folder>_]++
*:collapse-folder* [_<folder>_]
	Expands or collapses a folder when the directory tree is enabled. If no
	_<folder>_ argument is specified, the currently selected folder is acted
	upon.

*:export-mbox* _<file>_
	Exports messages in the current folder to an mbox file. If there are marked
	messages in the folder, only the marked ones are exported. Otherwise the
	whole folder is exported.

*:import-mbox* _<path>_
	Imports all messages from an (gzipped) mbox file to the current folder.
	_<path>_ can either be a path to a file or an URL.

	Examples:

	```
	:import-mbox ~/messages.mbox
	:import-mbox https://lists.sr.ht/~rjarry/aerc-devel/patches/55634/mbox
	:import-mbox https://lore.kernel.org/all/20190807155524.5112-1-steve.capper@arm.com/t.mbox.gz
	```

*:next-result*++
*:prev-result*
	Selects the next or previous search result.

*:query* [*-a* _<account>_] [*-n* _name_] [*-f*] _<notmuch query>_
	Create a virtual folder using the specified top-level notmuch query. This
	command is exclusive to the notmuch backend.

	*-a* _<account>_
		Change to _<folder>_ of _<account>_ and focus its corresponding
		tab.

	*-n* _<name>_
		Specify the display name for the virtual folder. If not provided,
		_<notmuch query>_ is used as the display name.

	*-f*
		Load the query results into an already existing folder (messages
		in the original folder are not deleted).

*:search* [_<options>_] _<terms>_...
	Searches the current folder for messages matching the given set of
	conditions.  The search syntax is dependent on the underlying backend.
	Refer to *aerc-search*(1) for details.

*:select* _<n>_++
*:select-message* _<n>_
	Selects the _<n>_\th message in the message list (and scrolls it into
	view if necessary).

*:hsplit* [[_+_|_-_]_<n>_]
*:split* [[_+_|_-_]_<n>_]
	Creates a horizontal split, showing _<n>_ messages and a message view
	below the message list. If a _+_ or _-_ is prepended, the message list
	size will grow or shrink accordingly. The split can be cleared by
	calling *:[h]split* _0_, or just *:[h]split*. The split can be toggled
	by calling split with the same (absolute) size repeatedly. For example,
	*:[h]split* _10_ will create a split. Calling *:[h]split* _10_ again
	will remove the split. If not specified, _<n>_ is set to an estimation
	based on the user's terminal. Also see *:vsplit*.

*:sort* [[*-r*] _<criterion>_]...
	Sorts the message list by the given criteria. *-r* sorts the
	immediately following criterion in reverse order.

	Available criteria:

[[ *Criterion*
:- *Description*
|  _arrival_
:- Date and time of the messages arrival
|  _cc_
:- Addresses in the Cc field
|  _date_
:- Date and time of the message
|  _from_
:- Addresses in the From field
|  _read_
:- Presence of the read flag
|  _flagged_
:- Presence of the flagged flag
|  _size_
:- Size of the message
|  _subject_
:- Subject of the message
|  _to_
:- Addresses in the To field

*:toggle-threads*
	Toggles between message threading and the normal message list.

*:fold* [*-at*]++
*:unfold* [*-at*]
	Collapse or un-collapse the thread children of the selected message.
	If the toggle flag *-t* is set, the folded status is changed. If the
	*-a* flag is set, all threads in the current view are affected. Folded
	threads can be identified by _{{.Thread\*}}_ template attributes
	in *[ui].index-columns*. See *aerc-config*(5) and *aerc-templates*(7)
	for more details.

*:toggle-thread-context*
	Toggles between showing entire thread (when supported) and only showing
	messages which match the current query / mailbox.

*:view* [*-pb*]++
*:view-message* [*-pb*]
	Opens the message viewer to display the selected message. If the peek
	flag *-p* is set, the message will not be marked as seen and ignores the
	*auto-mark-read* config. If the background flag *-b* is set, the message
	will be opened in a background tab.

*:vsplit* [[_+_|_-_]_<n>_]
	Creates a vertical split of the message list. The message list will be
	_<n>_ columns wide, and a vertical message view will be shown to the
	right of the message list. If a _+_ or _-_ is prepended, the message
	list size will grow or shrink accordingly. The split can be cleared by
	calling *:vsplit* _0_, or just *:vsplit*. The split can be toggled by
	calling split with the same (absolute) size repeatedly. For example,
	*:vsplit* _10_ will create a split. Calling *:vsplit* _10_ again will
	remove the split. If not specified, _<n>_ is set to an estimation based
	on the user's terminal. Also see *:split*.

## MESSAGE VIEW COMMANDS

*:close*
	Closes the message viewer.

*:next* _<n>_[_%_]++
*:prev* _<n>_[_%_]
	Selects the next (or previous) message in the message list. If specified as
	a percentage, the percentage is applied to the number of messages shown on
	screen and the cursor advances that far.

*:next-part*++
*:prev-part*
	Cycles between message parts being shown. The list of message parts is shown
	at the bottom of the message viewer.

*:open* [*-d*] [_<args...>_]
	Saves the current message part to a temporary file, then opens it. If no
	arguments are provided, it will open the current MIME part with the
	matching command in the *[openers]* section of _aerc.conf_. When no match
	is found in *[openers]*, it falls back to the default system handler.

	*-d*: Delete the temporary file after the opener exits

	When arguments are provided:

	- The first argument must be the program to open the message part with.
	  Subsequent args are passed to that program.
	- _{}_ will be expanded as the temporary filename to be opened. If it is
	  not encountered in the arguments, the temporary filename will be
	  appened to the end of the command.

*:open-link* _<url>_ [_<args...>_]
	Open the specified URL with an external program. The opening logic is
	the same than for *:open* but the opener program will be looked up
	according to the URL scheme MIME type: _x-scheme-handler/<scheme>_.

*:save* [*-fpaA*] _<path>_
	Saves the current message part to the given path.
	If the path is not an absolute path, *[general].default-save-path* from
	_aerc.conf_ will be prepended to the path given.
	If path ends in a trailing slash or if a folder exists on disc or if *-a*
	is specified, aerc assumes it to be a directory.
	When passed a directory *:save* infers the filename from the mail part if
	possible, or if that fails, uses _aerc\_$DATE_.

	*-f*: Overwrite the destination whether or not it exists

	*-p*: Create any directories in the path that do not exist

	*-a*: Save all attachments. Individual filenames cannot be specified.

	*-A*: Same as *-a* but saves all the named parts, not just attachments.

*:mark* [*-atvT*]
	Marks messages. Commands will execute on all marked messages instead of the
	highlighted one if applicable. The flags below can be combined as needed.

	*-a*: Apply to all messages in the current folder

	*-t*: toggle the mark state instead of marking a message

	*-v*: Enter / leave visual mark mode

	*-V*: Same as *-v* but does not clear existing selection

	*-T*: Marks the displayed message thread of the selected message.

*:unmark* [*-at*]
	Unmarks messages. The flags below can be combined as needed.

	*-a*: Apply to all messages in the current folder

	*-t*: toggle the mark state instead of unmarking a message

*:remark*
	Re-select the last set of marked messages. Can be used to chain commands
	after a selection has been acted upon

*:toggle-headers*
	Toggles the visibility of the message headers.

*:toggle-key-passthrough*
	Enter or exit the *[view::passthrough]* key bindings context. See
	*aerc-binds*(5) for more details.

## MESSAGE COMPOSE COMMANDS

*:abort*
	Close the composer without sending, discarding the message in progress.

	If the text editor exits with an error (e.g. *:cq* in *vim*(1)), the
	message is immediately discarded.

*:attach* _<path>_++
*:attach* *-m* [_<arg>_]++
*:attach* *-r* <name> <cmd>
	Attaches the file at the given path to the email. The path can contain
	globbing syntax described at https://godocs.io/path/filepath#Match.

	*-m* [_<arg>_]
		Runs the *file-picker-cmd* to select files to be attached.
		Requires an argument when *file-picker-cmd* contains the _%s_ verb.

	*-r* <name> <cmd>
		Runs the <cmd>, reads its output and attaches it as <name>. The
		attachment MIME type is derived from the <name>'s extension.

*:attach-key*
	Attaches the public key for the configured account to the email.

*:detach* [_<path>_]
	Detaches the file with the given path from the composed email. If no path is
	specified, detaches the first attachment instead. The path can contain
	globbing syntax described at https://godocs.io/path/filepath#Match.

*:cc* _<addresses>_++
*:bcc* _<addresses>_
	Sets the Cc or Bcc header to the given addresses. If an editor for the header
	is not currently visible in the compose window, a new one will be added.

*:edit* [*-e*|*-E*]
	(Re-)opens your text editor to edit the message in progress. This will
	also allow editing the message headers. Only available from the review
	screen.

	*-e*: Forces *[compose].edit-headers* = _true_ for this message only.

	*-E*: Forces *[compose].edit-headers* = _false_ for this message only.

*:multipart* [*-d*] _<mime/type>_
	Makes the message to multipart/alternative and add the specified
	_<mime/type>_ part. Only the MIME types that are configured in the
	*[multipart-converters]* section of _aerc.conf_ are supported and their
	related commands will be used to generate the alternate part.

	*-d*:
		Remove the specified alternative _<mime/type>_ instead of
		adding it. If no alternative parts are left, make the message
		text/plain (i.e. not multipart/alternative).

*:next-field*++
*:prev-field*
	Cycles between input fields in the compose window. Only available when
	the text editor is visible and *[compose].edit-headers* = _false_.

*:postpone* [*-t* _<folder>_]
	Saves the current state of the message to the *postpone* folder (from
	_accounts.conf_) for the current account by default. Only available from
	the review screen.

	*-t*: Overrides the target folder for saving the message

	If the message was force-recalled with *:recall -f* from a different folder,
	the *:postpone* command will save it back to that folder instead of the
	default *postpone* folder configured in settings. Use *-t* to override that
	or use *:mv* to move the saved message to a different folder.

*:send* [*-a* _<scheme>_] [*-t* _<folder>_]
	Sends the message using this accounts default outgoing transport
	configuration. For details on configuring outgoing mail delivery consult
	*aerc-accounts*(5). Only available from the review screen.

	*-a*: Archive the message being replied to. See *:archive* for schemes.

	*-t*: Overrides the Copy-To folder for saving the message.

*:switch-account* _<account-name>_++
*:switch-account* *-n*++
*:switch-account* *-p*
	Switches the account. Can be used to switch to a specific account from
	its name or to cycle through accounts using the *-p* and *-n* flags.

	*-p*: switch to previous account

	*-n*: switch to next account

*:header* [*-f*] _<name>_ [_<value>_]
*:header* [*-d*] _<name>_
	Add a new email header to the compose window. If the header is already
	set and is not empty, *-f* must be used to overwrite its value.

	*-f*: Overwrite any existing header.

	*-d*: Remove the header instead of adding it.

*:encrypt*
	Encrypt the message to all recipients. If a key for a recipient cannot
	be found the message will not be encrypted.

*:sign*
	Sign the message using the account's default key. If *pgp-key-id* is set
	in _accounts.conf_ (see *aerc-accounts*(5)), it will be used in
	priority. Otherwise, the *From* header address will be used to look for
	a matching private key in the pgp keyring.

## TERMINAL COMMANDS

*:close*
	Closes the terminal.

# LOGGING

Aerc does not log by default, but collecting log output can be useful for
troubleshooting and reporting issues. Redirecting stdout when invoking aerc will
write log messages to that file:

	$ aerc > aerc.log

Persistent logging can be configured via the *log-file* and *log-level* settings
in _aerc.conf_.

# SEE ALSO

*aerc-config*(5) *aerc-imap*(5) *aerc-jmap*(5) *aerc-notmuch*(5) *aerc-smtp*(5)
*aerc-maildir*(5) *aerc-sendmail*(5) *aerc-search*(1) *aerc-stylesets*(7)
*aerc-templates*(7) *aerc-accounts*(5) *aerc-binds*(5) *aerc-tutorial*(7)
*aerc-patch*(7)

# AUTHORS

Originally created by Drew DeVault and maintained by Robin Jarry who is assisted
by other open source contributors. For more information about aerc development,
see _https://sr.ht/~rjarry/aerc/_.
