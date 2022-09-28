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
	imap[s][+insecure|+oauthbearer|+xoauth2]://username[:password]@hostname[:port]?[:oauth2_params]

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

		If specified and a token_endpoint is provided, the configured password
		is used as a refresh token to obtain an access token. If token_endpoint
		is omitted, refresh token exchange is skipped, and the password acts
		like an access token instead.

		- token_endpoint (optional)
		- client_id (optional)
		- client_secret (optional)
		- scope (optional)

		Example:
		imaps+oauthbearer://...?token_endpoint=https://...&client_id=

	*imaps+xoauth2://*
		IMAP with TLS/SSL using XOAUTH2 Authentication. Parameters are
		the same as OAUTHBEARER.

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

*check-mail-include*
	Specifies the comma separated list of folders to include when checking for
	new mail with *check-mail*. Names prefixed with ~ are interpreted as regular
	expressions.

	Default: all folders

*check-mail-exclude*
	Specifies the comma separated list of folders to exclude when checking for
	new mail with *check-mail*. Names prefixed with ~ are interpreted as regular
	expressions.
	Note that this overrides anything from *check-mail-include*.

	Default: no folders

*cache-headers*
	If set to true, headers will be cached. The cached headers will be stored
	in $XDG_CACHE_HOME/aerc, which defaults to ~/.cache/aerc.

	Default: false

*cache-max-age*
	Defines the maximum age of cached files. Note: the longest unit of time
	cache-max-age can be specified in is hours. Set to 0 to disable cleaning
	the cache

	Default: 720h (30 days)

*idle-timeout*
	The length of time the client will wait for the server to send any final
	update before the IDLE is closed.

	Default: 10ms

*idle-debounce*
	Specifies the length of time from the last client command until the
	idler starts.

	Default: 10ms

# SEE ALSO

*aerc*(1) *aerc-config*(5)

# AUTHORS

Originally created by Drew DeVault <sir@cmpwn.com> and maintained by Robin
Jarry <robin@jarry.cc> who is assisted by other open source contributors. For
more information about aerc development, see https://sr.ht/~rjarry/aerc/.
