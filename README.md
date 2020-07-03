# aerc

aerc is an email client for your terminal.

A short demonstration can be found on [https://aerc-mail.org/](https://aerc-mail.org/)

Join the IRC channel: [#aerc on irc.freenode.net](http://webchat.freenode.net/?channels=aerc&uio=d4)
for end-user support, and [#aerc-dev](http://webchat.freenode.net/?channels=aerc-dev&uio=d4)
for development.

## Building

Install the dependencies:

- go (>=1.13)
- [scdoc](https://git.sr.ht/~sircmpwn/scdoc)

Then compile aerc:

    $ make

aerc optionally supports notmuch. To enable it, you need to have a recent
version of [notmuch](https://notmuchmail.org/#index7h2), including the header
files (notmuch.h). Then compile aerc with the necessary build tags:

    $ GOFLAGS=-tags=notmuch make

## Installation

    # make install
    $ aerc

On its first run, aerc will copy the default config files to `~/.config/aerc`
on Linux or `~/Library/Preferences/aerc` on MacOS and show the account
configuration wizard.

If you redirect stdout to a file, logging output will be written to that file:

    $ aerc > log

## Resources

[Send patches](https://git-send-email.io) and questions to
[~sircmpwn/aerc@lists.sr.ht](https://lists.sr.ht/~sircmpwn/aerc).

Subscribe to release announcements on
[~sircmpwn/aerc-announce](https://lists.sr.ht/~sircmpwn/aerc-announce)

Bugs & todo here: [~sircmpwn/aerc2](https://todo.sr.ht/~sircmpwn/aerc2)
