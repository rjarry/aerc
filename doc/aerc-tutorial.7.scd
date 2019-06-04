aerc-tutorial(7)

# NAME

aerc - the world's best email client

# INTRODUCTION

Welcome to aerc! This tutorial will guide you through your first steps in using
the client. This tutorial is a man page - you can read it again later with
*:help tutorial* from aerc, or *man aerc-tutorial* from your terminal.

First, let's introduce some basic keybindings. For convention, we'll use *<C-p>*
to represent Ctrl+p, which matches the convention used for writing keybindings
for aerc.

*<C-p>*, *<C-n>*
	Cycles to the previous or next tab

Try using these now to switch between your message list and the tutorial. In
your message list, we use vim-style keys to get around.

*k*, *j*
	Scrolls up and down between messages

*<C-u>*, *<C-d>*
	Scrolls half a page up or down

*g*, *G*
	Selects the first or last message, respectively

*K*, *J*
	Switches between folders in the sidebar

*<Enter>*
	Opens the selected message

# THE MESSAGE VIEWER

Press *<Enter>* to open a message. By default, the message viewer will display
your message using *less*(1). This should also have familiar, vim-like
keybindings for scrolling around in your message.

Multipart messages (messages with attachments, or messages with several
alternative formats) show a part selector on the bottom of the message viewer.

*<C-k>*, *<C-j>*
	Cycle between parts of a multipart message

*q*
	Close the message viewer

# COMPOSING MESSAGES

Return to the message list by pressing *q* to dismiss the message viewer. Once
there, let's compose a message.

*C*
	Compose a new message

*rr*
	Reply-all to a message

*rq*
	Reply-all to a message, and pre-fill the editor with a quoted version of the
	message being replied to

*Rr*
	Reply to a message

*Rq*
	Reply to a message, and pre-fill the editor with a quoted version of the
	message being replied to

For now, let's use *C* to compose a new message. The message composer will
appear. You should see To, From, and Subject lines, as well as your *$EDITOR*.
You can use *<Tab>* or *<C-j>* and *<C-k>* to cycle between these fields (tab
won't cycle between fields once you enter the editor, but *<C-j>* and *<C-k>*
will).

Let's send an email to yourself. Note that the To and From headers only accept
RFC 5322 addresses, e.g. *John Doe <john@example.org>*, or simply
*<john@example.org>*. Separate multiple recipients with commas. Go ahead and
fill out an email (to yourself, perhaps?), then close the editor. You can add
additional headers like Cc and Reply-To by simply adding them to the top of your
email, adding a blank line between the email's headers and body.

The message review screen is shown next. You have a chance now to revise the
email before it's sent. Press *y* to send the email if it looks good.

*Note*: when using the terminal in the message view, you can summon aerc's ex
command line by using *<C-x>*. ':' is sent to the editor.

# USING THE TERMINAL

aerc comes with an embedded terminal, which you've already used to view and edit
emails. We can also use this for other purposes, such as referencing a git
repository while reviewing a patch. From the message list, we can use the
following keybindings to open a terminal:

*<C-t>*
	Opens a new terminal tab, running your shell

*$*, *!*
	Prompts for a command to run, then opens a new terminal tab running that
	command

*|*
	Prompts for a command to run, then pipes the selected email into that
	command and displays the result on a new terminal tab

Try pressing *$* and entering "top". You can also use the *:cd* command to
change aerc's working directory, and the directory in which new terminals run.
Use *:pwd* to see it again if you're not sure where you are.

# ADDITIONAL NOTES

## COMMANDS

Every keybinding is ultimately bound to an aerc command. You can also summon the
command line by pressing ':', then entering one of these commands. See *aerc*(1)
or *:help* for a full list of commands.

## MESSAGE FILTERS

When displaying messages in the message viewer, aerc will pipe them through a
_message filter_ first. This allows you to decode messages in non-plaintext
formats, add syntax highlighting, etc. aerc ships with a few default filters:

- Emails which begin with "[PATCH" will be piped into a filter for rendering
  git-format-patch and hg export emails.
- text/html is rendered with w3m in a network sandbox
- text/\* is rendered with a simple filter for coloring quoted text

## CUSTOMIZING AERC

Aerc is highly customizable. Review *aerc-config*(5) (or use *:help config*) to
learn more about how to add custom keybindings, install new message filters,
change its appearance and behavior, and so on.

# AUTHORS

Maintained by Drew DeVault <sir@cmpwn.com>, who is assisted by other open
source contributors. For more information about aerc development, see
https://git.sr.ht/~sircmpwn/aerc.
