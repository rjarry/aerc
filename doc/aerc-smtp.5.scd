AERC-SMTP(5)

# NAME

aerc-smtp - SMTP configuration for *aerc*(1)

# SYNOPSIS

aerc implements the SMTP protocol as specified by RFC 5321.

# CONFIGURATION

SMTP configuration may be done interactively with the *:new-account* command.

In _accounts.conf_ (see *aerc-accounts*(5)), the following SMTP-specific options
are available:

*outgoing* = _<scheme>_[_+<auth>_]://_<username>_[_:<password>_]_@<hostname>_[_:<port>_]?[_<oauth2_params>_]
	Remember that all fields must be URL encoded. The _@_ symbol, when URL
	encoded, is _%40_.

	The value of _<scheme>_ can be:

	_smtp_
		SMTP with STARTTLS

	_smtp+insecure_
		SMTP without STARTTLS

	_smtps_
		SMTP with TLS/SSL

	Additionally, you can specify an authentication mechanism like so:

	_none_
		No authentication is required to use this SMTP server. You may omit the
		username and password in this case.

	_plain_
		Authenticate with a username and password using AUTH PLAIN. This is the
		default behavior.

	_login_
		Authenticate with a username and password using AUTH LOGIN. This is an obsolete
		protocol, but is required for some common webmail providers.

	_oauthbearer_
		SMTP with TLS/SSL using OAUTHBEARER Authentication. See
		documentation in *aerc-imap*(5) for usage.

	_xoauth2_
		SMTP with TLS/SSL using XOAUTH2 Authentication. See
		documentation in *aerc-imap*(5) for usage.

*outgoing-cred-cmd* = _<command>_
	Specifies the command to run to get the password for the SMTP
	account. This command will be run using _sh -c [command]_. If a
	password is specified in the *outgoing* option, the password will
	take precedence over this command.

	Example:
		outgoing-cred-cmd = pass hostname/username

*smtp-domain* = _<domain>_
	Local domain name to use in the HELO/EHLO SMTP command. Set this to a fully
	qualified domain name if the server requires it as an antispam measure.

	Default: _localhost_

# SEE ALSO

*aerc*(1) *aerc-accounts*(5)

# AUTHORS

Originally created by Drew DeVault <sir@cmpwn.com> and maintained by Robin
Jarry <robin@jarry.cc> who is assisted by other open source contributors. For
more information about aerc development, see https://sr.ht/~rjarry/aerc/.
