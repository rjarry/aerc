aerc(1)

# NAME

aerc - the world's best email client

# SYNOPSIS

_aerc_ [-v] [-a <account-name[,account-name]>] [mailto:...]

For a guided tutorial, use *:help tutorial* from aerc, or *man aerc-tutorial*
from your terminal.

# OPTIONS

*-v*
	Prints the installed version of aerc and exits.

*-a <account-name[,account-name]>*
	Load only the named accounts, as opposed to all configured accounts.
	List must be comma separated, with no spaces. The account order will be
	preserved.

*mailto:address[,address][?query[&query]]*
	Opens the composer with the address(es) in the "to" field. These
	addresses must not be percent encoded.
	If _aerc_ is already running, the composer is started in this instance,
	otherwise _aerc_ will be started.
	The following (optional) query parameters are supported:

[[ *Query*
:[ *Description*
|  subject=<text>
:  Subject line will be completed with the text
|  body=<text>
:  Message body will be completed with the text
|  cc=<address[,address]>
:  CC header will be completed with the list of addresses
|  bcc=<address[,address]>
:  BCC header will be completed with the list of addresses
|  in-reply-to=<\<message-id\>>
:  In-reply-to header will be set to the message id
|  account=<accountname>
:  Specify the account (must be in accounts.conf; default is the selected account)

	Note that reserved characters in the queries must be percent encoded.

# RUNTIME COMMANDS

To execute a command, press ':' to bring up the command interface. Commands may
also be bound to keys, see *aerc-config*(5) for details. In some contexts, such
as the terminal emulator, '<c-x>' is used to bring up the command interface.

Different commands work in different contexts, depending on the kind of tab you
have selected.

Aerc stores a history of commands, which can be cycled through in command mode.
Pressing the up key cycles backwards in history, while pressing down cycles
forwards.

## GLOBAL COMMANDS

These commands work in any context.

*cd* <directory>
	Changes aerc's current working directory.

*z* <directory or zoxide query>
	Changes aerc's current working directory using zoxide. If zoxide is not on
	*$PATH*., the command will not be registered.

*change-tab* [+|-]<tab name or index>
*ct* [+|-]<tab name or index>
	Changes the focus to the tab with the given name. If a number is given,
	it's treated as an index. If + or - is specified, the number is interpreted
	as a delta from the selected tab.

*exec* <command...>
	Executes an arbitrary command in the background. Aerc will set the
	environment variables *$account* and *$folder* when the command is
	executed from an Account tab or an opened message.

	*Note*: commands executed in this way are not executed with the shell.

*eml* [<path>]
	Opens an eml file and displays the message in the message viewer.

*pwd*
	Displays aerc's current working directory in the status bar.

*set* <category>.<option> <value>
	Changes the value of a configuration parameter at runtime. The category is
	the name of the [heading] for the config option you wish to change, and the
	option is the name of the config option. For example, to change this option:

	\[ui]++
	index-format=before

	Use :set ui.index-format after.

*term* [command...]
*terminal*
	Opens a new terminal tab with a shell running in the current working
	directory, or the specified command.

*move-tab* [+|-]<index>
	Moves the selected tab to the given index. If + or - is specified, the
	number is interpreted as a delta from the selected tab.

*prev-tab* [n], *next-tab* [n]
	Cycles to the previous or next tab in the list, repeating n times
	(default: 1).

*pin-tab*
	Moves the current tab to the left of all non-pinned tabs and displays
	the *pinned-tab-marker* (default: `) to the left of the tab title.

*unpin-tab*
	Removes the *pinned-tab-marker* from the current tab and returns the tab
	to its previous location.

*prompt* <prompt> <command...>
	Displays the prompt on the status bar, waits for user input, then appends
	that input as the last argument to the command and executes it. The input is
	passed as one argument to the command, unless it is empty, in which case no
	extra argument is added.

*choose* -o <key> <text> <command> [-o <key> <text> <command>]...
	Prompts the user to choose from various options.

*quit* [-f]
*exit* [-f]
	Exits aerc. If a task is being performed that should not be interrupted
	(like sending a message), a normal quit call might fail. In this case,
	closing aerc can be forced with the -f option.

## MESSAGE COMMANDS

These commands are valid in any context that has a selected message (e.g. the
message list, the message in the message viewer, etc).

*archive* <scheme>
	Moves the selected message to the archive. The available schemes are:

	*flat*: No special structure, all messages in the archive directory

	*year*: Messages are stored in folders per year

	*month*: Messages are stored in folders per year and subfolders per month

*accept*
	Accepts an iCalendar meeting invitation.

*accept-tentative*
	Accepts an iCalendar meeting invitation tentatively.

*copy* <target>
*cp* <target>
	Copies the selected message to the target folder.

*decline*
	Declines an iCalendar meeting invitation.

*delete*
*delete-message*
	Deletes the selected message.

*envelope [-h] [-s <format-specifier>]*
	Opens the message envelope in a dialog popup.

	*-h*: Show all header fields

	*-s* <format-specifier>
		User-defined format specifier requiring two %s for the key and
		value strings. Default format: '%-20.20s: %s'

*recall* [-f]
	Opens the selected message for re-editing. Messages can only be
	recalled from the postpone directory. The original message is deleted.

	*-f*: Open the message for re-editing even if it is not in the postpone
	directory. The original message will be deleted only if it is in the
	postpone directory.

*forward* [-A | -F] [-T <template-file>] [address...]
	Opens the composer to forward the selected message to another recipient.

	*-A*: Forward the message and all attachments.

	*-F*: Forward the full message as an RFC 2822 attachment.

	*-T* <template-file>
		Use the specified template file for creating the initial
		message body. Unless *-F* is specified, this defaults to what
		is set as _forwards_ in the _[templates]_ section of
		_aerc.conf_.

*move* <target>
*mv* <target>
	Moves the selected message to the target folder.

*pipe* [-bmp] <cmd>
	Downloads and pipes the selected message into the given shell command, and
	opens a new terminal tab to show the result. By default, the selected
	message part is used in the message viewer and the full message is used in
	the message list.

	Operates on multiple messages when they are marked. When piping multiple
	messages, aerc will write them with mbox format separators.

	*-b*: Run the command in the background instead of opening a terminal tab

	*-m*: Pipe the full message

	*-p*: Pipe just the selected message part, if applicable

*reply* [-aq] [-T <template-file>]
	Opens the composer to reply to the selected message.

	*-a*: Reply all

	*-q*: Insert a quoted version of the selected message into the reply editor

	*-T* <template-file>
		Use the specified template file for creating the initial
		message body. If *-q* is specified, defaults to what is set as
		_quoted-reply_ in the _[templates]_ section of _aerc.conf_.

*read* [-t]
	Marks the marked or selected messages as read.

	*-t*: Toggle the messages between read and unread.

*unread* [-t]
	Marks the marked or selected messages as unread.

	*-t*: Toggle the messages between read and unread.

*flag* [-t] [-a | -x <flag>]
	Sets (enables) a certain flag on the marked or selected messages.

	*-t*: Toggle the flag instead of setting (enabling) it.

	*-a*: Mark message as answered/unanswered.

	*-x <flag>*: Mark message with specific flag.

	The available flags are (adapted from RFC 3501, section 2.3.2):

	Seen
		Message has been read
	Answered
		Message has been answered
	Flagged
		Message is flagged for urgent/special attention

*unflag* [-t] <flag>
	Operates exactly like *flag*, defaulting to unsetting (disabling) flags.

*modify-labels* <[+-]label>...
*tag* <[+-]label>...
	Modify message labels (e.g. notmuch tags). Labels prefixed with a '+' are
	added, those prefixed with a '-' removed. As a convenience, labels without
	either operand add the specified label.

	Example: `modify-labels +inbox -spam unread` adds the labels inbox and unread
	and removes spam

*unsubscribe*
	Attempt to automatically unsubscribe the user from the mailing list through
	use of the List-Unsubscribe header. If supported, aerc may open a compose
	window pre-filled with the unsubscribe information or open the unsubscribe
	URL in a web browser.

## MESSAGE LIST COMMANDS

*clear* [-s]
	Clears the current search or filter criteria.

	By default, the selected message will be kept. To clear the selected
	message and move cursor to the top of the message list, use the -s flag.

	*-s*
		Selects the message at the top of the message list after clearing.

*cf* <folder>
	Change the folder shown in the message list.

*check-mail*
	Check for new mail on the selected account. Non-imap backends require
	check-mail-cmd to be set in order for aerc to initiate a check for new mail.
	Issuing a manual :check-mail command will reset the timer for automatic checking.

*compose* [-H] [<body>] [-T <template-file>]
	Open the compose window to send a new email. The new email will be sent with
	the current account's outgoing transport configuration. For details on
	configuring outgoing mail delivery consult *aerc-config*(5).

	*-H* <header>
		Add the specified header to the message, e.g. 'compose -H "X-Custom: custom
		value"'
	*-T* <template-file>
		Use the specified template file for creating the initial
		message body.

*filter* [options] <terms...>
	Similar to *search*, but filters the displayed messages to only the search
	results. See the documentation for *search* for more details.

*mkdir* <name>
	Creates a new folder for this account and changes to that folder.

*rmdir* [-f]
	Removes the current folder.

	By default, it will fail if the directory is non-empty (see *-f*).

	*-f*
		Remove the directory even if it contains messages.

	Some programs that sync maildirs may recover deleted directories (e.g.
	offlineimap).  These can either be specially configured to properly
	handle directory deletion, or special commands need to be run to delete
	directories (e.g. 'offlineimap --delete-folder').

	It is possible, with a slow connection and the 'imap' backend, that new
	messages arrive in the directory before they show up - using 'rmdir' at
	this moment would delete the directory and such new messages before the
	user sees them.

*next* <n>[%], *prev* <n>[%]
*next-message* <n>[%], *prev-message* <n>[%]
	Selects the next (or previous) message in the message list. If specified as
	a percentage, the percentage is applied to the number of messages shown on
	screen and the cursor advances that far.

*next-folder* <n>, *prev-folder* <n>
	Cycles to the next (or previous) folder shown in the sidebar, repeated n
	times (default: 1).

*expand-folder*, *collapse-folder*
	Expands or collapses the current folder when the directory tree is
	enabled.

*export-mbox* <file>
	Exports all messages in the current folder to an mbox file.

*import-mbox* <file>
	Imports all messages from an mbox file to the current folder.

*next-result*, *prev-result*
	Selects the next or previous search result.

*search*
	Searches the current folder.
	The search syntax is dependent on the underlying backend.
	Refer to *aerc-search*(1) for details

*select* <n>
*select-message* <n>
	Selects the nth message in the message list (and scrolls it into view if
	necessary).

*split* [+|-] [<n>]
	Creates a horizontal split, showing a <n> messages and a message view
	below the message list. If a + or - is prepended, the message list size
	will grow or shrink accordingly. The split can be cleared by calling
	:split 0, or just :split. The split can be toggled by calling split with
	the same size repeatedly. For example, :split 10 will create a split.
	Calling :split 10 again will remove the split. Also see *vsplit*

*sort* [[-r] <criterion>]...
	Sorts the message list by the given criteria. *-r* sorts the
	immediately following criterion in reverse order.

	Available criteria:

[[ *Criterion*
:- *Description*
|  arrival
:- Date and time of the messages arrival
|  cc
:- Addresses in the "cc" field
|  date
:- Date and time of the message
|  from
:- Addresses in the "from" field
|  read
:- Presence of the read flag
|  size
:- Size of the message
|  subject
:- Subject of the message
|  to
:- Addresses in the "to" field

*toggle-threads*
	Toggles between message threading and the normal message list.

*view* [-p]
*view-message* [-p]
	Opens the message viewer to display the selected message. If the peek
	flag -p is set, the message will not be marked as "seen" and ignores the
	"auto-mark-read" config.

*vsplit* [+|-] [<n>]
	Creates a vertical split of the message list. The message list will be
	<n> columns wide, and a vertical message view will be shown to the right
	of the message list. If a + or - is prepended, the message list size
	will grow or shrink accordingly. The split can be cleared by calling
	:vsplit 0, or just :vsplit. The split can be toggled by calling split
	with the same size repeatedly. For example, :vsplit 10 will create a
	split. Calling :vsplit 10 again will remove the split. Also see *split*

## MESSAGE VIEW COMMANDS

*close*
	Closes the message viewer.

*next* <n>[%], *prev* <n>[%]
	Selects the next (or previous) message in the message list. If specified as
	a percentage, the percentage is applied to the number of messages shown on
	screen and the cursor advances that far.

*next-part*, *prev-part*
	Cycles between message parts being shown. The list of message parts is shown
	at the bottom of the message viewer.

*open* [args...]
	Saves the current message part to a temporary file, then opens it. If no
	arguments are provided, it will open the current MIME part with the
	matching command in the *[openers]* section of _aerc.conf_. When no match
	is found in *[openers]*, it falls back to the default system handler.

	When arguments are provided:

	- The first argument must be the program to open the message part with.
	  Subsequent args are passed to that program.
	- *{}* will be expanded as the temporary filename to be opened. If it is
	  not encountered in the arguments, the temporary filename will be
	  appened to the end of the command.

*save* [-fpa] <path>
	Saves the current message part to the given path.
	If the path is not an absolute path, general.default-save-path will be
	prepended to the path given.
	If path ends in a trailing slash or if a folder exists on disc or if -a
	is specified, aerc assumes it to be a directory.
	When passed a directory :save infers the filename from the mail part if
	possible, or if that fails, uses "aerc_$DATE".

	*-f*: Overwrite the destination whether or not it exists

	*-p*: Create any directories in the path that do not exist

	*-a*: Save all attachments. Individual filenames cannot be specified.

*mark* [-atvT]
	Marks messages. Commands will execute on all marked messages instead of the
	highlighted one if applicable. The flags below can be combined as needed.

	*-a*: Apply to all messages in the current folder

	*-t*: toggle the mark state instead of marking a message

	*-v*: Enter / leave visual mark mode

	*-V*: Same as -v but does not clear existing selection

	*-T*: Marks the displayed message thread of the selected message.

*unmark* [-at]
	Unmarks messages. The flags below can be combined as needed.

	*-a*: Apply to all messages in the current folder

	*-t*: toggle the mark state instead of unmarking a message

*remark*
	Re-select the last set of marked messages. Can be used to chain commands
	after a selection has been acted upon

## MESSAGE COMPOSE COMMANDS

*abort*
	Close the composer without sending, discarding the message in progress.

*attach* <path>
	Attaches the file at the given path to the email. The path can contain
	globbing syntax described at https://godocs.io/path/filepath#Match.

*attach-key*
	Attaches the public key for the configured account to the email.

*detach* [path]
	Detaches the file with the given path from the composed email. If no path is
	specified, detaches the first attachment instead.

*cc* [addresses], *bcc* [addresses]
	Sets the Cc or Bcc header to the given addresses. If an editor for the header
	is not currently visible in the compose window, a new one will be added.

*edit*
	(Re-) opens your text editor to edit the message in progress.

*next-field*, *prev-field*
	Cycles between input fields in the compose window.

*postpone*
	Saves the current state of the message to the *postpone* folder for the
	current account.

*save* [-p] <path>
	Saves the selected message part to the specified path. If -p is selected,
	aerc will create any missing directories in the specified path. If the path
	specified is a directory or ends in /, aerc will use the attachment filename
	if available or a generated name if not.

*send*
	Sends the message using this accounts default outgoing transport
	configuration. For details on configuring outgoing mail delivery consult
	*aerc-config*(5).

*switch-account* <-n|-p|account-name>
	Switches the account. Can be used to switch to a specific account from
	its name or to cycle through accounts using the -p and -n flags.

	*-p*: switch to previous account
	*-n*: switch to next account

*header* [-f] <field> [value]
	Add a new email header. If the header already exists, -f must be
	specified to replace the given value.

*toggle-headers*
	Toggles the visibility of the message headers.

## TERMINAL COMMANDS

*close*
	Closes the terminal.

# LOGGING

Aerc does not log by default, but collecting log output can be useful for
troubleshooting and reporting issues. Redirecting stdout when invoking aerc will
write log messages to that file:

	$ aerc > log

# SEE ALSO

*aerc-config*(5) *aerc-imap*(5) *aerc-smtp*(5) *aerc-maildir*(5)
*aerc-sendmail*(5) *aerc-tutorial*(7)

# AUTHORS

Originally created by Drew DeVault <sir@cmpwn.com> and maintained by Robin
Jarry <robin@jarry.cc> who is assisted by other open source contributors. For
more information about aerc development, see https://sr.ht/~rjarry/aerc/.
