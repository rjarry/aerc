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
	imap[s][+insecure|+oauthbearer]://username[:password]@hostname[:port]?[:oauth2_params]

	Remember that all fields must be URL encoded. The "@" symbol, when URL
	encoded, is *%40*.

	The meaning of the scheme component is:

	*imap://*:
		IMAP with STARTTLS

	*imap+insecure://*:
		IMAP without STARTTLS

	*imaps*:
		IMAP with TLS/SSL

	*imaps+oauthbearer://*
		IMAP with TLS/SSL using OAUTHBEARER Authentication

		*oauth2_params:*

		If specified, the configured password is used as an refresh token that
		is exchanged with an access token

		- token_endpoint (required)
		- client_id (optional)
		- client_secret (optional)
		- scope (optional)

		Example:
		imaps+oauthbearer://...?token_endpoint=https://...&client_id=

*source-cred-cmd*
	Specifies the command to run to get the password for the IMAP
	account. This command will be run using `sh -c [command]`. If a
	password is specified in the *source* option, the password will
	take precedence over this command.

	Example:

	pass hostname/username

*connection-timeout*
	Maximum delay to establish a connection to the IMAP server. See
	https://pkg.go.dev/time#ParseDuration.

	Default: 30s

*keepalive-period*
	The interval between the last data packet sent (simple ACKs are not
	considered data) and the first keepalive probe. After the connection is
	marked to need keepalive, this counter is not used any further. See
	https://pkg.go.dev/time#ParseDuration.

	By default, the system tcp socket settings are used.

*keepalive-probes*
	The number of unacknowledged probes to send before considering the
	connection dead and notifying the application layer.

	By default, the system tcp socket settings are used.
	If keepalive-period is specified, this option defaults to 3 probes.

	This option is only supported on linux. On other platforms, it will be
	ignored.

*keepalive-interval*
	The interval between subsequential keepalive probes, regardless of what
	the connection has exchanged in the meantime. Fractional seconds are
	truncated.

	By default, the system tcp socket settings are used.
	If keepalive-period is specified, this option defaults to 3s.

	This option is only supported on linux. On other platforms, it will be
	ignored.

# SEE ALSO

*aerc*(1) *aerc-config*(5)

# AUTHORS

Originally created by Drew DeVault <sir@cmpwn.com> and maintained by by Robin
Jarry <robin@jarry.cc> who is assisted by other open source contributors. For
more information about aerc development, see https://sr.ht/~rjarry/aerc/.
