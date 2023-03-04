AERC(1)

# NAME

aerc - a pretty good email client.

# SYNOPSIS

*aerc* [*-v*] [*-a* _<account>_[,_<account>_]] [*mailto:*_..._]

For a guided tutorial, use *:help tutorial* from aerc, or *man aerc-tutorial*
from your terminal.

# OPTIONS

*-v*
	Prints the installed version of aerc and exits.

*-a* _<account>_[,_<account>_]
	Load only the named accounts, as opposed to all configured accounts.
	List must be comma separated, with no spaces. The account order will be
	preserved.

*mailto:*_address[,address][?query[&query]]_
	Opens the composer with the address(es) in the To field. These
	addresses must not be percent encoded.
	If aerc is already running, the composer is started in this instance,
	otherwise aerc will be started.
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

	Note that reserved characters in the queries must be percent encoded.

\:*command* [...]
	Run an aerc-internal command as you would in Ex-Mode. See *RUNTIME
	COMMANDS* below.

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

*:cd* _<directory>_
	Changes aerc's current working directory.

*:z* _<directory or zoxide query>_
	Changes aerc's current working directory using zoxide. If zoxide is not on
	*$PATH*., the command will not be registered.

*:change-tab* [*+*|*-*]_<tab name or index>_++
*:ct* [*+*|*-*]_<tab name or index>_
	Changes the focus to the tab with the given name. If a number is given,
	it's treated as an index. If *+* or *-* is specified, the number is interpreted
	as a delta from the selected tab.

*:exec* _<command>_
	Executes an arbitrary command in the background. Aerc will set the
	environment variables *$account* and *$folder* when the command is
	executed from an Account tab or an opened message.

	Note: commands executed in this way are not executed with the shell.

*:eml* [_<path>_]++
*:preview*
	Opens an eml file and displays the message in the message viewer.

	Can also be used in the message viewer to open an rfc822 attachment or
	in the composer to preview the message.

*:pwd*
	Displays aerc's current working directory in the status bar.

*:term* [_<command>..._]++
*:terminal*
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

*:choose* *-o* _<key>_ _<text>_ _<command>_ [*-o* _<key>_ _<text>_ _<command>_]...
	Prompts the user to choose from various options.

*:quit* [*-f*]++
*:exit* [*-f*]
	Exits aerc. If a task is being performed that should not be interrupted
	(like sending a message), a normal quit call might fail. In this case,
	closing aerc can be forced with the *-f* option.

## MESSAGE COMMANDS

These commands are valid in any context that has a selected message (e.g. the
message list, the message in the message viewer, etc).

*:archive* _<scheme>_
	Moves the selected message to the archive. The available schemes are:

	_flat_: No special structure, all messages in the archive directory

	_year_: Messages are stored in folders per year

	_month_: Messages are stored in folders per year and subfolders per month

*:accept*
	Accepts an iCalendar meeting invitation.

*:accept-tentative*
	Accepts an iCalendar meeting invitation tentatively.

*:copy* _<target>_++
*:cp* _<target>_
	Copies the selected message to the target folder.

*:decline*
	Declines an iCalendar meeting invitation.

*:delete*++
*:delete-message*
	Deletes the selected message.

*:envelope* [*-h*] [*-s* _<format-specifier>_]
	Opens the message envelope in a dialog popup.

	*-h*: Show all header fields

	*-s* _<format-specifier>_
		User-defined format specifier requiring two _%s_ for the key and
		value strings. Default format: _%-20.20s: %s_

*:recall* [*-f*]
	Opens the selected message for re-editing. Messages can only be
	recalled from the postpone directory. The original message is deleted.

	*-f*: Open the message for re-editing even if it is not in the postpone
	directory. The original message will be deleted only if it is in the
	postpone directory.

*:forward* [*-A*|*-F*] [*-T* _<template-file>_] [_<address>_...]
	Opens the composer to forward the selected message to another recipient.

	*-A*: Forward the message and all attachments.

	*-F*: Forward the full message as an RFC 2822 attachment.

	*-T* _<template-file>_
		Use the specified template file for creating the initial
		message body. Unless *-F* is specified, this defaults to what
		is set as *forwards* in the *[templates]* section of
		_aerc.conf_.

*:move* _<target>_++
*:mv* _<target>_
	Moves the selected message to the target folder.

*:pipe* [*-bmp*] _<cmd>_
	Downloads and pipes the selected message into the given shell command, and
	opens a new terminal tab to show the result. By default, the selected
	message part is used in the message viewer and the full message is used in
	the message list.

	Operates on multiple messages when they are marked. When piping multiple
	messages, aerc will write them with mbox format separators.

	*-b*: Run the command in the background instead of opening a terminal tab

	*-m*: Pipe the full message

	*-p*: Pipe just the selected message part, if applicable

	This can be used to apply patch series with git:

		*:pipe -m* _git am -3_

	When at least one marked message subject matches a patch series (e.g.
	_[PATCH X/Y]_), all marked messages will be sorted by subject to ensure
	that the patches are applied in order.

*:reply* [*-acq*] [*-T* _<template-file>_]
	Opens the composer to reply to the selected message.

	*-a*: Reply all

	*-c*: Close the view tab when replying. If the reply is not sent, reopen
	the view tab.

	*-q*: Insert a quoted version of the selected message into the reply editor

	*-T* _<template-file>_
		Use the specified template file for creating the initial
		message body. If *-q* is specified, defaults to what is set as
		*quoted-reply* in the *[templates]* section of _aerc.conf_.

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
		_Flagged_
			Message is flagged for urgent/special attention

*:unflag* [*-t*] _<flag>_
	Operates exactly like *:flag*, defaulting to unsetting (disabling) flags.

*:modify-labels* [_+_|_-_]_<label>_...++
*:tag* [_+_|_-_]_<label>_...
	Modify message labels (e.g. notmuch tags). Labels prefixed with a *+* are
	added, those prefixed with a *-* removed. As a convenience, labels without
	either operand add the specified label.

	Example: add _inbox_ and _unread_ labels, remove _spam_ label.

		*:modify-labels* _+inbox_ _-spam_ _unread_

*:unsubscribe*
	Attempt to automatically unsubscribe the user from the mailing list through
	use of the List-Unsubscribe header. If supported, aerc may open a compose
	window pre-filled with the unsubscribe information or open the unsubscribe
	URL in a web browser.

## MESSAGE LIST COMMANDS

*:clear* [*-s*]
	Clears the current search or filter criteria.

	By default, the selected message will be kept. To clear the selected
	message and move cursor to the top of the message list, use the *-s* flag.

	*-s*: Selects the message at the top of the message list after clearing.

*:cf* _<folder>_
	Change the folder shown in the message list.

*:check-mail*
	Check for new mail on the selected account. Non-imap backends require
	check-mail-cmd to be set in order for aerc to initiate a check for new mail.
	Issuing a manual *:check-mail* command will reset the timer for automatic checking.

*:compose* [*-H* _"<header>: <value>"_] [*-T* _<template-file>_] [_<body>_]
	Open the compose window to send a new email. The new email will be sent with
	the current account's outgoing transport configuration. For details on
	configuring outgoing mail delivery consult *aerc-accounts*(5).

	*-H* _"<header>: <value>"_
		Add the specified header to the message, e.g:

			*:compose -H* _"X-Custom: custom value"_

	*-T* _<template-file>_
		Use the specified template file for creating the initial
		message body.

	_<body>_: The initial message body.

*:filter* [_<options>_] _<terms>_...
	Similar to *:search*, but filters the displayed messages to only the search
	results. See the documentation for *:search* for more details.

*:mkdir* _<name>_
	Creates a new folder for this account and changes to that folder.

*:rmdir* [*-f*]
	Removes the current folder.

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

*:expand-folder*++
*:collapse-folder*
	Expands or collapses the current folder when the directory tree is
	enabled.

*:export-mbox* _<file>_
	Exports all messages in the current folder to an mbox file.

*:import-mbox* _<file>_
	Imports all messages from an mbox file to the current folder.

*:next-result*++
*:prev-result*
	Selects the next or previous search result.

*:filter* [_<options>_] _<terms>_...
	Searches the current folder.
	The search syntax is dependent on the underlying backend.
	Refer to *aerc-search*(1) for details

*:select* _<n>_++
*:select-message* _<n>_
	Selects the _<n>_\th message in the message list (and scrolls it into
	view if necessary).

*:split* [[_+_|_-_]_<n>_]
	Creates a horizontal split, showing _<n>_ messages and a message view
	below the message list. If a _+_ or _-_ is prepended, the message list
	size will grow or shrink accordingly. The split can be cleared by
	calling *:split* _0_, or just *:split*. The split can be toggled by
	calling split with the same (absolute) size repeatedly. For example,
	*:split* _10_ will create a split. Calling *:split* _10_ again will
	remove the split. If not specified, _<n>_ is set to an estimation based
	on the user's terminal. Also see *:vsplit*.

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
|  _size_
:- Size of the message
|  _subject_
:- Subject of the message
|  _to_
:- Addresses in the To field

*:toggle-threads*
	Toggles between message threading and the normal message list.

*:view* [*-p*]++
*:view-message* [*-p*]
	Opens the message viewer to display the selected message. If the peek
	flag *-p* is set, the message will not be marked as seen and ignores the
	*auto-mark-read* config.

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

*:open* [_<args...>_]
	Saves the current message part to a temporary file, then opens it. If no
	arguments are provided, it will open the current MIME part with the
	matching command in the *[openers]* section of _aerc.conf_. When no match
	is found in *[openers]*, it falls back to the default system handler.

	When arguments are provided:

	- The first argument must be the program to open the message part with.
	  Subsequent args are passed to that program.
	- _{}_ will be expanded as the temporary filename to be opened. If it is
	  not encountered in the arguments, the temporary filename will be
	  appened to the end of the command.

*:save* [*-fpa*] _<path>_
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

## MESSAGE COMPOSE COMMANDS

*:abort*
	Close the composer without sending, discarding the message in progress.

*:attach* _<path>_++
*:attach* *-m* [_<arg>_]
	Attaches the file at the given path to the email. The path can contain
	globbing syntax described at https://godocs.io/path/filepath#Match.

	*-m* [_<arg>_]
		Runs the *file-picker-cmd* to select files to be attached.
		Requires an argument when *file-picker-cmd* contains the _%s_ verb.

*:attach-key*
	Attaches the public key for the configured account to the email.

*:detach* [_<path>_]
	Detaches the file with the given path from the composed email. If no path is
	specified, detaches the first attachment instead.

*:cc* _<addresses>_++
*:bcc* _<addresses>_
	Sets the Cc or Bcc header to the given addresses. If an editor for the header
	is not currently visible in the compose window, a new one will be added.

*:edit*
	(Re-)opens your text editor to edit the message in progress.

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
	Cycles between input fields in the compose window.

*:postpone*
	Saves the current state of the message to the *postpone* folder (from
	_accounts.conf_) for the current account.

*:save* [*-p*] _<path>_
	Saves the selected message part to the specified path. If *-p* is selected,
	aerc will create any missing directories in the specified path. If the path
	specified is a directory or ends in _/_, aerc will use the attachment filename
	if available or a generated name if not.

*:send* [*-a* _<scheme>_]
	Sends the message using this accounts default outgoing transport
	configuration. For details on configuring outgoing mail delivery consult
	*aerc-accounts*(5).

	*-a*: Archive the message being replied to. See *:archive* for schemes.

*:switch-account* _<account-name>_++
*:switch-account* *-n*++
*:switch-account* *-p*
	Switches the account. Can be used to switch to a specific account from
	its name or to cycle through accounts using the *-p* and *-n* flags.

	*-p*: switch to previous account

	*-n*: switch to next account

*:header* [*-f*] _<name>_ [_<value>_]
	Add a new email header. If the header already exists, *-f* must be
	specified to replace the given value.

*:toggle-headers*
	Toggles the visibility of the message headers.

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

*aerc-config*(5) *aerc-imap*(5) *aerc-smtp*(5) *aerc-maildir*(5)
*aerc-sendmail*(5) *aerc-tutorial*(7)

# AUTHORS

Originally created by Drew DeVault <sir@cmpwn.com> and maintained by Robin
Jarry <robin@jarry.cc> who is assisted by other open source contributors. For
more information about aerc development, see https://sr.ht/~rjarry/aerc/.
