CARDDAV-QUERY(1)

# NAME

carddav-query - Query a CardDAV server for contact names and emails.

# SYNOPSIS

*carddav-query* [*-h*] [*-l* _<limit>_] [*-v*] [*-c* _<file>_]
\[*-s* _<section>_] [*-k* _<key\_source>_] [*-C* _<key\_cred\_cmd>_]
\[*-s* _<server\_url>_] [*-u* _<username>_] [*-p* _<password>_] _<term>_ [_<term>_ ...]

This tool has been tailored for use as *address-book-cmd* in *aerc-config*(5).

# OPTIONS

*-h*, *--help*
	show this help message and exit

*-v*, *--verbose*
	Print debug info on stderr.

*-l* _<limit>_, *--limit* _<limit>_
	Maximum number of results returned by the server. If the server does not
	support limiting, this option will be disregarded.

	Default: _10_

*-c* _<file>_, *--config-file* _<file>_
	INI configuration file from which to read the CardDAV URL endpoint.

	Default: _~/.config/aerc/accounts.conf_

*-S* _<section>_, *--config-section* _<section>_
	INI configuration section where to find _<key\_source>_ and
	_<key\_cred\_cmd>_. By default the first section where _<key\_source>_
	is found will be used.

*-k* _<key\_source>_, *--config-key-source* _<key\_source>_
	INI configuration key to lookup in _<section>_ from _<file>_. The value
	must respect the following format:

		https?://_<username>_[:_<password>_]@_<hostname>_/_<path/to/addressbook>_

	Both _<username>_ and _<password>_ must be percent encoded. If
	_<password>_ is omitted, it can be provided via *--config-key-cred-cmd*
	or *--password*.

	Default: _carddav-source_

*-C* _<key\_cred\_cmd>_, *--config-key-cred-cmd* _<key\_cred\_cmd>_
	INI configuration key to lookup in _<section>_ from _<file>_. The value
	is a command that will be executed with *sh -c* to determine
	_<password>_ if it is not present in _<key\_source>_.

	Default: _carddav-source-cred-cmd_

*-s* _<server_url>_, *--server-url* _<server_url>_
	CardDAV server URL endpoint. Overrides configuration file.

*-u* _<username>_, *--username* _<username>_
	Username to authenticate on the server. Overrides configuration file.

*-p* _<password>_, *--password* _<password>_
	Password for the specified user. Overrides configuration file.

# POSITIONAL ARGUMENTS

_<term>_
	Search term. Will be used to search contacts from their FN (formatted
	name), EMAIL, NICKNAME, ORG (company) and TITLE fields.

# EXAMPLES

These are excerpts of _~/.config/aerc/accounts.conf_.

## Fastmail

```
[fastmail]
carddav-source = https://janedoe%40fastmail.com@carddav.fastmail.com/dav/addressbooks/user/janedoe@fastmail.com/Default
carddav-source-cred-cmd = pass fastmail.com/janedoe
address-book-cmd = carddav-query -S fastmail %s
```

## Gmail

```
[gmail]
carddav-source = https://johndoe%40gmail.com@www.googleapis.com/carddav/v1/principals/johndoe@gmail.com/lists/default
carddav-source-cred-cmd = pass gmail.com/johndoe
address-book-cmd = carddav-query -S gmail %s
```

# SEE ALSO

*aerc-config*(5)

# AUTHORS

Created by Robin Jarry <robin@jarry.cc> who is assisted by other open source
contributors. For more information about aerc development, see
https://sr.ht/~rjarry/aerc/.
