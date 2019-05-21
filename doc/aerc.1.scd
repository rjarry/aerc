aerc(1)

# NAME

aerc - the world's best email client

# SYNOPSIS

_aerc_

For a guided tutorial, use *:help tutorial*.

# RUNTIME COMMANDS

To execute a command, press ':' to bring up the command interface. Commands may
also be bound to keys, see *aerc-config*(5) for details. In some contexts, such
as the terminal emulator, ';' is used to bring up the command interface.

Different commands work in different contexts, depending on the kind of tab you
have selected.

## GLOBAL COMMANDS

These commands work in any context.

*cd* <directory>
	Changes aerc's current working directory.

*pwd*
	Displays aerc's current working directory in the status bar.

*term* [command...]
	Opens a new terminal tab with a shell running in the current working
	directory, or the specified command.

*prev-tab* [n], *next-tab* [n]
	Cycles to the previous or next tab in the list, repeating n times
	(default: 1).

*quit*
	Exits aerc.

## MESSAGE LIST COMMANDS

*cf* <folder>
	Change the folder shown in the message list.

*compose*
	Open the compose window to send a new email. The new email will be sent with
	the current account's outgoing transport configuration, see
	*aerc-config*(5) for details on configuring outgoing emails.

*copy* <target>
	Copies the selected message to the target folder.

*delete-message*
	Deletes the selected message.

*move* <target>
	Moves the selected message to the target folder.

*next-folder* <n>, *prev-folder* <n>
	Cycles to the next (or previous) folder shown in the sidebar, repeated n
	times (default: 1).

*next-message* <n>[%], *prev-message* <n>[%]
	Selects the next (or previous) message in the message list. If specified as
	a percentage, the percentage is applied to the number of messages shown on
	screen and the cursor advances that far.

*pipe* <cmd>
	Downloads and pipes the selected message into the given shell command, and
	opens a new terminal tab to show the result.

*reply* [-aq]
	Opens the composer to reply to the selected message.

	*-a*: Reply all

	*-q*: Insert a quoted version of the selected message into the reply editor

*select-message* <n>
	Selects the nth message in the message list (and scrolls it into view if
	necessary).

*view-message*
	Opens the message viewer to display the selected message.

## MESSAGE VIEW COMMANDS

*close*
	Closes the message viewer.

## TERMINAL COMMANDS

*close*
	Closes the terminal.

# SEE ALSO

*aerc-config*(5) *aerc-imap*(5) *aerc-smtp*(5)

# AUTHORS

Maintained by Drew DeVault <sir@cmpwn.com>, who is assisted by other open
source contributors. For more information about aerc development, see
https://git.sr.ht/~sircmpwn/aerc.
