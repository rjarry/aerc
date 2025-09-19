---
title: "aerc-wiki: Integrations/sieve-connect"
---

# sieve-connect

[sieve-connect](https://github.com/philpennock/sieve-connect) is a client for
the MANAGESIEVE protocol written in Perl. For convenience, you can add a
keybind to Aerc to open sieve-connect logged in to MANAGESIEVE.

Firstly install sieve-connect.

Create a script somewhere on your system (mine is just in my `~/scripts`
directory):

```sh
#!/bin/sh

sieve-connect -s hostname -u username --passwordfd=5 5<<<"$(secret-tool lookup Title 'main email')"
```

Replace `hostname` with the MANAGESIEVE server hostname. Replace `username`
with your username.

The rest of the command programmatically gets our password so that we can log
in automatically. Omit it if you want to manually type in your password every
time.

The `passwordfd` flag is set to a file descriptor into which we provide the
password. In this example, we are using the `secret-tool` tool to get our
password, but you can use any other method of getting your password
programmatically.

Save the script and then edit `~/.config/aerc/binds.conf`. Add a global
binding to this script, e.g.

```conf
<C-s> = :term /home/username/scripts/sieve_connect<Enter>
```

Now you can do Ctrl-S in Aerc to open a new tab to manage your Sieve scripts.

