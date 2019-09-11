aerc(1)

# NAME

aerc - the world's best email client

# SYNOPSIS

_aerc_ [-v]

For a guided tutorial, use *:help tutorial* from aerc, or *man aerc-tutorial*
from your terminal.

# OPTIONS

*-v*
	Prints the installed version of aerc and exits.

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

*change-tab* [+|-]<tab name or index>
	Changes the focus to the tab with the given name. If a number is given,
	it's treated as an index. If + or - is specified, the number is interpreted
	as a delta from the selected tab.

*exec* <command...>
	Executes an arbitrary command in the background.

	*Note*: commands executed in this way are not executed with the shell.

*pwd*
	Displays aerc's current working directory in the status bar.

*set* <category>.<option> <value>
	Changes the value of a configuration paramter at runtime. The category is
	the name of the [heading] for the config option you wish to change, and the
	option is the name of the config option. For example, to change this option:

	\[ui]++
	index-format=before

	Use :set ui.index-format after.

*term* [command...]
	Opens a new terminal tab with a shell running in the current working
	directory, or the specified command.

*prev-tab* [n], *next-tab* [n]
	Cycles to the previous or next tab in the list, repeating n times
	(default: 1).

*prompt* <prompt> <command...>
	Displays the prompt on the status bar, waits for user input, then appends
	that input as the last argument to the command and executes it. The input is
	passed as one argument to the command, unless it is empty, in which case no
	extra argument is added.

*quit*
	Exits aerc.

## MESSAGE COMMANDS

These commands are valid in any context that has a selected message (e.g. the
message list, the message in the message viewer, etc).

*archive* <scheme>
	Moves the selected message to the archive. The available schemes are:

	*flat*: No special structure, all messages in the archive directory

	*year*: Messages are stored in folders per year

	*month*: Messages are stored in folders per year and subfolders per month

*copy* <target>
	Copies the selected message to the target folder.

*delete*
	Deletes the selected message.

*forward* [-A] [address...]
	Opens the composer to forward the selected message to another recipient.

	*-A*: Forward the message as an RFC 8022 attachment.

*move* <target>
	Moves the selected message to the target folder.

*pipe* [-bmp] <cmd>
	Downloads and pipes the selected message into the given shell command, and
	opens a new terminal tab to show the result. By default, the selected
	message part is used in the message viewer and the full message is used in
	the message list.

	*-b*: Run the command in the background instead of opening a terminal tab

	*-m*: Pipe the full message

	*-p*: Pipe just the selected message part, if applicable

*reply* [-aq]
	Opens the composer to reply to the selected message.

	*-a*: Reply all

	*-q*: Insert a quoted version of the selected message into the reply editor

*read*
	Marks the selected message as read.

	*-t*: Toggle the selected message between read and unread.

*unread*
	Marks the selected message as unread.

	*-t*: Toggle the selected message between read and unread.

*modify-labels* <[+-]label>...
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

*clear*
	Clears the current search or filter criteria.

*cf* <folder>
	Change the folder shown in the message list.

*compose* [-H] [<body>]
	Open the compose window to send a new email. The new email will be sent with
	the current account's outgoing transport configuration. For details on
	configuring outgoing mail delivery consult *aerc-config*(5).

	*-H* <header>
		Add the specified header to the message, e.g. 'compose -H "X-Custom: custom
		value"'

*filter* [options] <terms...>
	Similar to *search*, but filters the displayed messages to only the search
	results. See the documentation for *search* for more details.

*mkdir* <name>
	Creates a new folder for this account and changes to that folder.

*next* <n>[%], *prev* <n>[%]
	Selects the next (or previous) message in the message list. If specified as
	a percentage, the percentage is applied to the number of messages shown on
	screen and the cursor advances that far.

*next-folder* <n>, *prev-folder* <n>
	Cycles to the next (or previous) folder shown in the sidebar, repeated n
	times (default: 1).

*next-result*, *prev-result*
	Selects the next or previous search result.

*search*
	Searches the current folder.
	The search syntax is dependant on the underlying backend.
	Refer to *aerc-search*(1) for details

*select* <n>
	Selects the nth message in the message list (and scrolls it into view if
	necessary).

*view*
	Opens the message viewer to display the selected message.

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

*open*
	Saves the current message part in a temporary file and opens it
	with the system handler.

*save* [-p] <path>
	Saves the current message part to the given path.

	If no path is given but general.default-save-path is set, the
	file will be saved there.

	*-p*: Make any directories in the path that do not exist

## MESSAGE COMPOSE COMMANDS

*abort*
	Close the composor without sending, discarding the message in progress.

*attach* <path>
	Attaches the file at the given path to the email.

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

*save* [-p] <path>
	Saves the selected message part to the specified path. If -p is selected,
	aerc will create any missing directories in the specified path. If the path
	specified is a directory or ends in /, aerc will use the attachment filename
	if available or a generated name if not.

*send*
	Sends the message using this accounts default outgoing transport
	configuration. For details on configuring outgoing mail delivery consult
	*aerc-config*(5).

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

Maintained by Drew DeVault <sir@cmpwn.com>, who is assisted by other open
source contributors. For more information about aerc development, see
https://git.sr.ht/~sircmpwn/aerc.
