---
title: "aerc-wiki: Integrations/PGP"
---

# PGP

To use GnuPG, simply set the following in `aerc.conf`.  Unless if you want to
tinker with the internal implementation, **it is highly recommended that you
use gpg or similar external implementations.**
```conf
[general]
pgp-provider=gpg
```

When the above configuration option is unset or set to `internal`,
aerc uses an internal OpenPGP implementation.

> At the moment internal PGP support is still in it's early stages and will
> likely change as time goes on. Tickets relating to this can be found in
> ~sircmpwn's tracker: [#353](https://todo.sr.ht/~sircmpwn/aerc2/353)
> [#354](https://todo.sr.ht/~sircmpwn/aerc2/354)
> [#355](https://todo.sr.ht/~sircmpwn/aerc2/355)
> [#357](https://todo.sr.ht/~sircmpwn/aerc2/357)
> [and more](https://todo.sr.ht/~sircmpwn/aerc2?search=label%3A%22pgp%22)

**Please be aware:** at the moment internal PGP support requires you to export
your private keys. Please ensure that your home directory is protected against
unauthorised access.

To create a PGP-Keyring and fill it with your keys from GPG, you can run the
following commands:

```shell
gpg --export >> ~/.local/share/aerc/keyring.asc
gpg --export-secret-keys >> ~/.local/share/aerc/keyring.asc
```
