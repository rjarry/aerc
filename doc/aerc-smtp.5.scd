aerc-smtp(5)

# NAME

aerc-smtp - SMTP configuration for *aerc*(1)

# SYNOPSIS

aerc implements the SMTP protocol as specified by RFC 5321.

# CONFIGURATION

SMTP configuration may be done interactively with the :new-account command.

In accounts.conf (see *aerc-config*(5)), the following SMTP-specific options are
available:

*outgoing*
	smtp[s][+plain|+none]://username[:password]@hostname[:port]

	Remember that all fields must be URL encoded. The "@" symbol, when URL
	encoded, is *%40*.

	The meaning of the scheme component is:

	*smtp://*:
		Unencrypted SMTP

	*smtps://*:
		SMTP with TLS/SSL

	Additionally, you can specify an authentication mechanism like so:

	*+none*:
		No authentication is required to use this SMTP server. You may omit the
		username and password in this case.

	*+plain*:
		Authenticate with a username and password using AUTH PLAIN. This is the
		default behavior.

	*+login*:
		Authenticate with a username and password using AUTH LOGIN. This is an obsolete
		protocol, but is required for some common webmail providers.

*outgoing-cred-cmd*
	Specifies the command to run to get the password for the SMTP
	account. This command will be run using `sh -c [command]`. If a
	password is specified in the *outgoing* option, the password will
	take precedence over this command.

	Example:

	pass hostname/username

*smtp-starttls*
	Set this to `yes` if the server uses STARTTLS. In that case you should also
	use *smtp* instead of *smtps*.

# SEE ALSO

*aerc*(1) *aerc-config*(5)

# AUTHORS

Maintained by Drew DeVault <sir@cmpwn.com>, who is assisted by other open
source contributors. For more information about aerc development, see
https://git.sr.ht/~sircmpwn/aerc.
