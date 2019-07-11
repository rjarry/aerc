aerc-maildir(5)

# NAME

aerc-maildir - maildir configuration for *aerc*(1)

# SYNOPSIS

aerc implements the maildir format.

# CONFIGURATION

Maildir accounts currently are not supported with the :new-account command and
must be added manually to the *aerc-config*(5) file.

The following maildir-specific options are available:

*source*
	maildir://path

	The *source* indicates the path to the directory containing your maildirs
	rather than one maildir specifically.

	The path portion of the URL following _maildir://_ must be either an absolute
	path prefixed by */* or a path relative to your home directory prefixed with
	*~*. For example:

		source = maildir:///home/me/mail

		source = maildir://~/mail

# SEE ALSO

*aerc*(1) *aerc-config*(5) *aerc-smtp*(5)

# AUTHORS

Maintained by Drew DeVault <sir@cmpwn.com>, who is assisted by other open
source contributors. For more information about aerc development, see
https://git.sr.ht/~sircmpwn/aerc.
