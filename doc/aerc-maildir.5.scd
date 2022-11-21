AERC-MAILDIR(5)

# NAME

aerc-maildir - maildir configuration for *aerc*(1)

# SYNOPSIS

aerc implements the maildir format.

# CONFIGURATION

Maildir accounts currently are not supported with the :new-account command and
must be added manually to the *aerc-config*(5) file.

The following maildir-specific options are available:

*check-mail-cmd*
	Command to run in conjunction with *check-mail* option.

	Example:
		check-mail-cmd = mbsync -a

	Default: none

*check-mail-timeout*
	Timeout for the *check-mail-cmd*. The command will be stopped if it does
	not complete in this interval and an error will be displayed. Increase from
	the default if repeated errors occur

	Default: 10s

*source*
	maildir://path

	The *source* indicates the path to the directory containing your maildirs
	rather than one maildir specifically.

	The path portion of the URL following _maildir://_ must be either an absolute
	path prefixed by */* or a path relative to your home directory prefixed with
	*~*. For example:

		source = maildir:///home/me/mail

		source = maildir://~/mail

	If your maildir is using the Maildir++ directory layout, you can use the
	_maildirpp://_ scheme instead:

		source = maildirpp:///home/me/mail

		source = maildirpp://~/mail

# SEE ALSO

*aerc*(1) *aerc-config*(5) *aerc-smtp*(5) *aerc-notmuch*(5)

# AUTHORS

Originally created by Drew DeVault <sir@cmpwn.com> and maintained by Robin
Jarry <robin@jarry.cc> who is assisted by other open source contributors. For
more information about aerc development, see https://sr.ht/~rjarry/aerc/.
