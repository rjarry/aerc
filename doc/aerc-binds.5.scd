AERC-BINDS(5)

# NAME

aerc-binds - key bindings configuration file format for *aerc*(1)

# SYNOPSIS

The _binds.conf_ file is used for configuring keybindings used in the aerc
interactive client. It is expected to be in your XDG config home plus _aerc_,
which defaults to _~/.config/aerc/binds.conf_. If the file does not exist, the
built-in default will be installed. An alternate file can be specified via the
_--binds-conf_ command line argument, see *aerc*(1).

This file is written in the ini format with key bindings defined as:

	*<key sequence>* = _<command>_

Where *<key sequence>* is the keystrokes pressed (in order) to invoke this
keybinding, and _<command>_ specifies keystrokes that aerc will simulate when
the keybinding is invoked. Generally this is used to execute commands, for
example:

	*rq* = _:reply -q<Enter>_

Pressing *r*, then *q*, will simulate typing in _:reply -q<Enter>_, and execute
*:reply -q* accordingly. It is also possible to invoke keybindings recursively
in a similar fashion.

You may configure different keybindings for different contexts by writing them
into different *[sections]* of the ini file.

# CONTEXTS

The available contexts are:

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

	To customize the description shown on the review screen, add a comment
	(_" # "_) at the end of the keybinding. Example:

		p = :postpone<Enter> # I'll work on it later

	The order in which bindings are defined is preserved on the review
	screen. If a non-default binding is not annotated it will be displayed
	without any description.

	To hide a binding from the review screen, explicitly annotate it with
	a _" # -"_ comment. Example:

		<C-e> = :encrypt<Enter> # -

*[terminal]*
	keybindings for terminal tabs

You may also configure account specific key bindings for each context:

*[context:account=*_AccountName_*]*
	keybindings for this context and account, where _AccountName_ is a
	regular expression that matches the account name you provided in _accounts.conf_.

Folder and context-specific bindings can be configured for message lists:

*[messages:folder=*_FolderName_*]*++
*[view:folder=*_FolderName_*]*++
*[compose:folder=*_FolderName_*]*++
*[compose::editor:folder=*_FolderName_*]*++
*[compose::review:folder=*_FolderName_*]*
	keybindings under this section will be specific to the folder that
	matches the regular expression _FolderName_.
	Keybindings from a folder specifier will take precedence over account specifiers

# EXAMPLES

```
[messages:account=Mailbox]
c = :cf path:mailbox/** and<space>

[compose::editor:account=Mailbox2]

[compose::editor:folder=aerc]
y = :send -t aerc

[messages:folder=Drafts]
<Enter> = :recall<Enter>

[messages:folder=Archive/\d+/.*]
gi = :cf Inbox<Enter>
...
```

You may also configure global keybindings by placing them at the beginning of
the file, before specifying any context-specific sections.

Parent keybindings can be erased in the context ones by specifying an "empty"
binding:

```
[compose::review]
a = :attach<space>
d = :detach<space>

[compose::review:account=no-attachments]
a =
d =
```

If you want to change the default Delete action to move to Trash instead of
deleting permanently:

```
d = :read<Enter>:move Trash<Enter>
#...
[messages:folder=Trash]
d = :choose -o y 'Permanently delete?' delete-message<Enter>
# And similarly for Spam
```

# SPECIAL OPTIONS

In addition of user defined key sequences, the following special options are
available in each binding context:

*$noinherit* = _true_|_false_
	If set to _true_, global keybindings will not be effective in this context.

	Default: _false_

*$ex* = _<key-stroke>_
	This can be set to a keystroke which will bring up the command input in this
	context.

	Default: _:_

*$complete* = _<key-stroke>_
	This can be set to a keystroke which will trigger command completion in
	this context for text inputs that support it.

	Default: _<tab>_

	Note: automatic command completion is disabled when simulating
	keystrokes and re-enabled at the end. When *[ui].completion-min-chars*
	is set to _manual_ (see *aerc-config*(5)), it is possible to end
	a keybinding with the completion key to explicitly display the
	completion menu. E.g.:

		*o* = _:cf<space><tab>_

# SUPPORTED KEYS

In addition to letters and some characters (e.g. *a*, *RR*, *gu*, *?*, *!*,
etc.), special keys may be specified in *<angle brackets>*. The syntax for
modified or special keys is:

	<C-A-S-key>

Where C is control, A is alt, S is shift, and key is the named key or character.

Valid key names are:

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
|  *<down>*
:  Down arrow
|  *<right>*
:  Right arrow
|  *<left>*
:  Left arrow
|  *<pgup>*
:  Page Up
|  *<pgdn>*
:  Page Down
|  *<home>*
:  Home
|  *<end>*
:  End
|  *<insert>*
:  Insert
|  *<delete>*
:  Delete
|  *<backspace>*
:  Backspace
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
|  *<esc>*
:  Escape

# SEE ALSO

*aerc*(1) *aerc-config*(5)

# AUTHORS

Originally created by Drew DeVault and maintained by Robin Jarry who is assisted
by other open source contributors. For more information about aerc development,
see _https://sr.ht/~rjarry/aerc/_.
