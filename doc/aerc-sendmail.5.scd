AERC-SENDMAIL(5)

# NAME

aerc-sendmail - sendmail configuration for *aerc*(1)

# SYNOPSIS

aerc can defer to sendmail for the delivery of outgoing messages.

# CONFIGURATION

In _accounts.conf_ (see *aerc-accounts*(5)), the following sendmail-specific
options are available:

*outgoing* = _</path/to/sendmail>_
	This should be set to the path to the sendmail binary you wish to use,
	which is generally _/usr/sbin/sendmail_. aerc will execute it with a list of
	recipients on the command line and pipe the message to deliver to stdin.

# SEE ALSO

*aerc*(1) *aerc-accounts*(5)

# AUTHORS

Originally created by Drew DeVault <sir@cmpwn.com> and maintained by Robin
Jarry <robin@jarry.cc> who is assisted by other open source contributors. For
more information about aerc development, see https://sr.ht/~rjarry/aerc/.
