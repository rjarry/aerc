AERC-BINDS(5)

# NAME

aerc-binds - key bindings configuration file format for *aerc*(1)

# SYNOPSIS

The _binds.conf_ file is used for configuring keybindings used in the aerc
interactive client. It is expected to be in your XDG config home plus _aerc_,
which defaults to _~/.config/aerc/binds.conf_. If the file does not exist, the
built-in default will be installed.

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

*[terminal]*
	keybindings for terminal tabs

You may also configure account specific key bindings for each context:

*[context:account=*_AccountName_*]*
	keybindings for this context and account, where _AccountName_ is a
	regular expression that matches the account name you provided in _accounts.conf_.

Folder-specific bindings can be configured for message lists:

*[messages:folder=*_FolderName_*]*
	keybindings under this section will be specific to the folder that
	matches the regular expression _FolderName_.
	Keybindings from a folder specifier will take precedence over account specifiers

Examples:

```
[messages:account=Mailbox]
c = :cf path:mailbox/** and<space>

[compose::editor:account=Mailbox2]

[messages:folder=Drafts]
<Enter> = :recall<Enter>

[messages:folder=Archive/\d+/.*]
gi = :cf Inbox<Enter>
...
```

You may also configure global keybindings by placing them at the beginning of
the file, before specifying any context-specific sections.

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

# SUPPORTED KEYS

In addition to letters and some characters (e.g. *a*, *RR*, *gu*, *?*, *!*,
etc.), special keys may be specified in *<angle brackets>*. The following
special keys are supported:

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
|  *<c-delete>*
:  Ctrl+Delete
|  *<a-delete>*
:  Alt+Delete
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
|  *<c-space>*
:  Ctrl+Space
|  *<a-space>*
:  Alt+Space
|  *<c-a>*
:  Ctrl+a
|  *<a-a>*
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

*aerc*(1) *aerc-config*(5)

# AUTHORS

Originally created by Drew DeVault and maintained by Robin Jarry who is assisted
by other open source contributors. For more information about aerc development,
see _https://sr.ht/~rjarry/aerc/_.
