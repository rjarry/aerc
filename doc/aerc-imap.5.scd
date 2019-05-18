aerc-imap(5)

# NAME

aerc-imap - IMAP configuration for *aerc*(1)

# SYNOPSIS

aerc implements the IMAP protocol as specified by RFC 3501, with the following
IMAP extensions:

- IDLE (RFC 2177)

# CONFIGURATION

IMAP configuration may be done interactively with the :new-account command.

In accounts.conf (see *aerc-config*(5)), the following IMAP-specific options are
available:

*source*
	imap[s][+insecure]://username[:password]@hostname[:port]

	Remember that all fields must be URL encoded. The "@" symbol, when URL
	encoded, is *%40*.

	The meaning of the scheme component is:

	*imap://*:
		IMAP with STARTTLS

	*imap+insecure://*:
		IMAP without STARTTLS

	*imaps*:
		IMAP with TLS/SSL

*source_cred_cmd*
    Specifies the command to run to get the password for the IMAP
    account. This command will be run using `sh -c [command]`. If a
    password is specified in the *source* option, the password will
    take precedence over this command.

    Example:

    `pass hostname/username`

# SEE ALSO

*aerc*(1) *aerc-config*(5) *aerc-smtp*(5)

# AUTHORS

Maintained by Drew DeVault <sir@cmpwn.com>, who is assisted by other open
source contributors. For more information about aerc development, see
https://git.sr.ht/~sircmpwn/aerc.
