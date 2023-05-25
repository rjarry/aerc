---
title: "aerc-wiki: Providers/ProtonMail"
---

# ProtonMail

Using ProtonMail with aerc is not trivial, as you will likely experience
issues with the `protonmail-bridge` certificate. The workaround is to
add this certificate to the root trust database. The certificate must be
exported from the bridge.

```text
Settings -> Advanced settings -> Export TLS certificates
```

Once you have saved the certificate on the system, run the following
command. Please note that this is tested on Arch Linux and Fedora, and
might not be valid on other distros.

```bash
sudo trust anchor --store ~/.config/protonmail/bridge/cert.pem
```

The bridge's default configuration of STARTTLS instead of SSL should
work with the following configuration

Example account configuration, using the default ports of
`protonmail-bridge`:

```ini
[Protonmail]
source        = imap://youraccount%40protonmail.com:yourprotonmailbridgepassword@127.0.0.1:1143
outgoing      = smtp://youraccount%40protonmail.com:yourprotonmailbridgepassword@127.0.0.1:1025
default       = INBOX
from          = Your Name <youraccount@protonmail.com>
copy-to       = Sent
#smtp-starttls = yes # uncomment if using aerc <= 0.14
```

The first time you run aerc with this configuration you can expect a
very long wait before anything shows up in your inbox. This will be
considerably faster on subsequent launches.
