AERC-MAILDIR(5)

# NAME

aerc-maildir - maildir configuration for *aerc*(1)

# SYNOPSIS

aerc implements the maildir format.

# CONFIGURATION

Basic Maildir configuration may be done interactively with the *:new-account*
command.

The following maildir-specific options are available:

*check-mail-cmd* = _<command>_
	Command to run in conjunction with *check-mail* option.

	Example:
		check-mail-cmd = mbsync -a

*check-mail-timeout* = _<duration>_
	Timeout for the *check-mail-cmd*. The command will be stopped if it does
	not complete in this interval and an error will be displayed. Increase from
	the default if repeated errors occur

	Default: 10s

*source* = _maildir_|_maildirpp_://_<path>_
	The *source* indicates the path to the directory containing your maildirs
	rather than one maildir specifically.

	The path portion of the URL following _maildir://_ must be either an absolute
	path prefixed by _/_ or a path relative to your home directory prefixed with
	*~*. For example:

		source = maildir:///home/me/mail

		source = maildir://~/mail

	If your maildir is using the Maildir++ directory layout, you can use the
	_maildirpp://_ scheme instead:

		source = maildirpp:///home/me/mail

		source = maildirpp://~/mail

# SEE ALSO

*aerc*(1) *aerc-accounts*(5) *aerc-smtp*(5) *aerc-notmuch*(5)

# AUTHORS

Originally created by Drew DeVault and maintained by Robin Jarry who is assisted
by other open source contributors. For more information about aerc development,
see _https://sr.ht/~rjarry/aerc/_.
