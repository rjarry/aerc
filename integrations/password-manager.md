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

2. Edit your `accounts.conf` to use `source-cred-cmd` and `outgoing-cred-cmd`
   to point to the script.
```ini
source-cred-cmd = ~/.config/aerc/scripts/wait-for-creds.sh Title "Mailaccount (Work)"
```

aerc will now wait for the credentials to become available (for you to unlock
you password manager) when starting.
