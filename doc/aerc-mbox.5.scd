AERC-MBOX(5)

# NAME

aerc-mbox - mbox configuration for *aerc*(1)

# SYNOPSIS

aerc implements the mbox file format, specifically the "mboxo" variant.

In addition to static configuration in _accounts.conf_, an _mbox:_ URL can be
given as a command-line argument (see aerc(1)).

Mailboxes created via _mbox:_ exist only in memory.  An mbox file is
effectively opened read-only; changes (including creating and deleting
messages) are not written back to the file.

# CONFIGURATION

The following mbox-specific options are available:

*source* = _mbox_:_<path>_
	The *source* indicates the path to a single mbox file or to a directory
	containing any number of mbox files with the _mbox_ extension.

	The remainder of the URL following _mbox:_ must be either an absolute path
	prefixed by _/_ or a path relative to your home directory prefixed with
	*~*. For example:

		source = mbox:/home/me/mail/archive.mbox

		source = mbox:~/mail/archive.mbox

# SEE ALSO

*aerc*(1) *aerc-accounts*(5) *aerc-smtp*(5)

# AUTHORS

Originally created by Drew DeVault and maintained by Robin Jarry who is assisted
by other open source contributors. For more information about aerc development,
see _https://sr.ht/~rjarry/aerc/_.
