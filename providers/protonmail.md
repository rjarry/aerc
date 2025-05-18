---
title: "aerc-wiki: Providers/ProtonMail"
---

# ProtonMail

## ProtonMail bridge certificate installation

Using ProtonMail with aerc is not trivial, since IMAP and SMTP access to the
mailbox involves the [ProtonMail Bridge](https://proton.me/mail/bridge).

Interacting with the bridge requires installing the bridge certificate on the
system.

### Instructions for Linux

Export the certificate from the Bridge:

```text
Settings -> Advanced settings -> Export TLS certificates
```

Once you have saved the certificate on the system, run the following
command:

```bash
sudo trust anchor --store ~/.config/protonmail/bridge/cert.pem
```

Please note that this is tested on Arch Linux and Fedora, and
might not be valid on other distributions.

### Instructions for MacOS

On MacOS, you only need to run this command once from the
[Bridge CLI](https://proton.me/support/bridge-cli-guide):

```text
cert install
```

## Configuration

The bridge's default configuration of STARTTLS instead of SSL should
work with the following configuration, that assumes the account was created on
protonmail.com (simply replace protonmail.com by proton.me if it was created on
proton.me):

```ini
[Protonmail]
source        = imap://youraccount%40protonmail.com:yourprotonmailbridgepassword@127.0.0.1:1143
outgoing      = smtp://youraccount%40protonmail.com:yourprotonmailbridgepassword@127.0.0.1:1025
default       = INBOX
from          = Your Name <youraccount@protonmail.com>
copy-to       = Sent
```

The first time you run aerc with this configuration you can expect a
very long wait before anything shows up in your inbox. This will be
considerably faster on subsequent launches.
