AERC-IMAP(5)

# NAME

aerc-imap - IMAP configuration for *aerc*(1)

# SYNOPSIS

aerc implements the IMAP protocol as specified by RFC 3501, with the following
IMAP extensions:

- IDLE (RFC 2177)
- LIST-STATUS (RFC 5819)
- X-GM-EXT-1 (Gmail)

# CONFIGURATION

Basic IMAP configuration may be done interactively with the *:new-account*
command.

In _accounts.conf_ (see *aerc-accounts*(5)), the following IMAP-specific options
are available:

*source* = _<scheme>_://_<username>_[_:<password>_]_@<hostname>_[_:<port>_]_?_[_<oauth2_params>_]
	Remember that all fields must be URL encoded. The _@_ symbol, when URL
	encoded, is _%40_.

	Possible values of _<scheme>_ are:

	_imap_
		IMAP with STARTTLS

	_imap+insecure_
		IMAP without STARTTLS

	_imaps_
		IMAP with TLS/SSL

	_imaps+insecure_
		IMAP with TLS/SSL, skipping certificate verification

	_imaps+oauthbearer_
		IMAP with TLS/SSL using OAUTHBEARER Authentication

		_<oauth2_params>_:

		If specified and a _token_endpoint_ is provided, the configured password
		is used as a refresh token to obtain an access token. If _token_endpoint_
		is omitted, refresh token exchange is skipped, and the password acts
		like an access token instead.

		- _token_endpoint_ (optional)
		- _client_id_ (optional)
		- _client_secret_ (optional)
		- _scope_ (optional)

		Example:
			imaps+oauthbearer://...?token_endpoint=https://...&client_id=

	_imaps+xoauth2_
		IMAP with TLS/SSL using XOAUTH2 Authentication. Parameters are
		the same as OAUTHBEARER.

*source-cred-cmd* = _<command>_
	Specifies the command to run to get the password for the IMAP
	account. This command will be run using _sh -c command_. If a
	password is specified in the *source* option, the password will
	take precedence over this command.

	Example:
		source-cred-cmd = pass hostname/username

*connection-timeout* = _<duration_>
	Maximum delay to establish a connection to the IMAP server. See
	https://pkg.go.dev/time#ParseDuration.

	Default: _90s_

*keepalive-period* = _<duration>_
	The interval between the last data packet sent (simple ACKs are not
	considered data) and the first keepalive probe. After the connection is
	marked to need keepalive, this counter is not used any further. See
	https://pkg.go.dev/time#ParseDuration.

	By default, the system tcp socket settings are used.

*keepalive-probes* = _<int>_
	The number of unacknowledged probes to send before considering the
	connection dead and notifying the application layer.

	By default, the system tcp socket settings are used.
	If keepalive-period is specified, this option defaults to 3 probes.

	This option is only supported on linux. On other platforms, it will be
	ignored.

*keepalive-interval* = _<duration>_
	The interval between subsequential keepalive probes, regardless of what
	the connection has exchanged in the meantime. Fractional seconds are
	truncated.

	By default, the system tcp socket settings are used.
	If *keepalive-period* is specified, this option defaults to _3s_.

	This option is only supported on linux. On other platforms, it will be
	ignored.

*check-mail-include* = _<folder1,folder2,folder3...>_
	Specifies the comma separated list of folders to include when checking for
	new mail with *:check-mail*. Names prefixed with _~_ are interpreted as regular
	expressions. This setting is ignored if your IMAP server supports the
	LIST-STATUS command, in which case all folders will be checked.

	By default, all folders are included.

*check-mail-exclude* = _<folder1,folder2,folder3...>_
	Specifies the comma separated list of folders to exclude when checking for
	new mail with *:check-mail*. Names prefixed with _~_ are interpreted as regular
	expressions. This setting is ignored if your IMAP server supports the
	LIST-STATUS command, in which case all folders will be checked.
	Note that this overrides anything from *check-mail-include*.

	By default, no folders are excluded.

*cache-headers* = _true_|_false_
	If set to _true_, headers will be cached. The cached headers will be stored
	in _$XDG_CACHE_HOME/aerc_, which defaults to _~/.cache/aerc_.

	Default: _false_

*cache-max-age* = _<duration>_
	Defines the maximum age of cached files. Note: the longest unit of time
	*cache-max-age* can be specified in is hours. Set to _0_ to disable cleaning
	the cache

	Default: _720h_ (30 days)

*idle-timeout* = _<duration>_
	The length of time the client will wait for the server to send any final
	update before the IDLE is closed.

	Default: _10s_

*idle-debounce* = _<duration>_
	Specifies the length of time from the last client command until the
	idler starts.

	Default: _10ms_

*expunge-policy* = _auto_|_low-to-high_|_stable_
	Specifies the deletion policy used when deleting multiple messages in
	one shot. _auto_ attempts to automatically detect it, and will be
	correct most of the times. _low-to-high_ specifies that the server
	deletes messages in increasing sequence number order (this is what GMail
	and FastMail do, and is correctly handled by the automatic detection).
	_stable_ specifies that the server does not mutate the sequence numbers
	it received (this is what Dovecot and WorkMail do, and is *not* reliably
	automatically detected).

	Default: _auto_

*debug-log-path* = _<path>_
	Specifies the path where to store low-level IMAP debug logs (this can
	be useful to debug interactions between aerc and IMAP servers). If this
	path is writable, aerc will create and log to one file per account,
	called `imap_debug_ACCOUNTNAME.log`.

	By default, low-level IMAP debug logs are not generated.

# SEE ALSO

*aerc*(1) *aerc-accounts*(5)

# AUTHORS

Originally created by Drew DeVault and maintained by Robin Jarry who is assisted
by other open source contributors. For more information about aerc development,
see _https://sr.ht/~rjarry/aerc/_.
