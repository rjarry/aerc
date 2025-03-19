---
title: "aerc-wiki: Configurations/mailto"
---

# mailto

For mailto links to work properly you either need a running instance of aerc or
a modification to the aerc.desktop file to include your terminal emulator of
choice.

Copy the aerc.desktop file to your local applications directory:

`$ mkdir -p ~/.local/share/applications`

`$ cp /usr/share/applications/aerc.desktop \
~/.local/share/applications/aerc.desktop`

Edit for your terminal, the two lines that need to be changed are:

`Terminal=true`

`Exec=aerc %u`

Here is an example aerc.desktop that uses the `foot` terminal

```desktop
[Desktop Entry]
Version=1.0
Name=aerc

GenericName=Mail Client
GenericName[de]=Email Client
Comment=Launches the aerc email client
Comment[de]=Startet den aerc Email-Client
Keywords=Email,Mail,IMAP,SMTP
Categories=Office;Network;Email;ConsoleOnly

Type=Application
Icon=utilities-terminal
Terminal=false
Exec=foot -e aerc %u
MimeType=x-scheme-handler/mailto
```
