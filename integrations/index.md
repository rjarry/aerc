# Integrations

## abook

To use abook with aerc, you can simply add the following line to your
`aerc.conf`.

```ini
address-book-cmd=abook --mutt-query "%s"
```

## Get Password from your Password manager

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

```
source-cred-cmd = ~/.config/aerc/scripts/wait-for-creds.sh Title "Mailaccount (Work)"
```

aerc will now wait for the credentials to become available (for you to unlock
you password manager) when starting.

## PGP

> At the moment PGP Support is still in it's early stages and will likely
> change as time goes on. Tickets relating to this can be found in
> ~sircmpwn's tracker: [#353](https://todo.sr.ht/~sircmpwn/aerc2/353)
> [#354](https://todo.sr.ht/~sircmpwn/aerc2/354)
> [#355](https://todo.sr.ht/~sircmpwn/aerc2/355)
> [#357](https://todo.sr.ht/~sircmpwn/aerc2/357)
> [and more](https://todo.sr.ht/~sircmpwn/aerc2?search=label%3A%22pgp%22)

**Please be aware:** at the moment PGP support requires you to export your
private keys. Please ensure that your home directory is protected against
unauthorised access.

To create a PGP-Keyring and fill it with your keys from GPG, you can run the
following commands:

```shell
gpg --export >> ~/.local/share/aerc/keyring.asc
gpg --export-secret-keys >> ~/.local/share/aerc/keyring.asc
```
