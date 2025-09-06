---
title: "aerc-wiki: Integrations/password-manager"
---

# Get Password from your Password manager

Requires:
- a password-manager supporting Freedesktop.org Secret Service integration
- secret-tool (usually provided by `libsecret` or a similar package)

Tested with:
- KeePassXC

1. create the following script:
```shell
#!/bin/sh

secret-tool lookup "$1" "$2"
# wait until the password is available
while [ $? != 0 ]; do
	secret-tool lookup "$1" "$2"
done
```

Note that the way `secret-tool lookup` works is with a key/value pair in
whatever secret store you're using. So you could store a secret with:

```console
$ secret-tool store --label='foo' bar baz
```

And retrieve it with:

```console
$ secret-tool lookup bar baz
```

The key is `bar` and the value is `baz`. Most likely you would want to use it
like:

```console
$ secret-tool store --label='main email' Title user@example.com
```

You would be prompted to enter the password upon entering this; the password is
not part of the command.

Normally, adding secrets etc is all done inside your password manager, however
if you are using something like gnome-keyring, you may find this method easier
to set the key and value you intend to look up.

2. Edit your `accounts.conf` to use `source-cred-cmd` and `outgoing-cred-cmd`
   to point to the script.
```ini
source-cred-cmd = ~/.config/aerc/scripts/wait-for-creds.sh Title "Mailaccount (Work)"
```

aerc will now wait for the credentials to become available (for you to unlock
you password manager) when starting.
