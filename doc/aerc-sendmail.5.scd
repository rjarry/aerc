aerc-sendmail(5)

# NAME

aerc-sendmail - sendmail configuration for *aerc*(1)

# SYNOPSIS

aerc can defer to sendmail for the delivery of outgoing messages.

# CONFIGURATION

In accounts.conf (see *aerc-config*(5)), the following sendmail-specific
options are available:

*outgoing*
	/path/to/sendmail

	This should be set to the path to the sendmail binary you wish to use,
	which is generally /usr/bin/sendmail. aerc will execute it with a list of
	recipients on the command line and pipe the message to deliver to stdin.

# SEE ALSO

*aerc*(1) *aerc-config*(5)

# AUTHORS

Originally created by Drew DeVault <sir@cmpwn.com> and maintained by by Robin
Jarry <robin@jarry.cc> who is assisted by other open source contributors. For
more information about aerc development, see https://sr.ht/~rjarry/aerc/.
