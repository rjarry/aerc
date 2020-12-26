# aerc

aerc is an email client for your terminal.

A short demonstration can be found on [https://aerc-mail.org/](https://aerc-mail.org/)

Join the IRC channel: [#aerc on irc.freenode.net](http://webchat.freenode.net/?channels=aerc&uio=d4)
for end-user support, and [#aerc-dev](http://webchat.freenode.net/?channels=aerc-dev&uio=d4)
for development.

## Usage

On its first run, aerc will copy the default config files to `~/.config/aerc`
on Linux or `~/Library/Preferences/aerc` on MacOS (or `$XDG_CONFIG_HOME/aerc` if set)
and show the account configuration wizard.

If you redirect stdout to a file, logging output will be written to that file:

    $ aerc > log

For instructions and documentation: see `man aerc` and further specific man
pages on there.

Note that the example HTML filter (off by default), additionally needs `w3m` and
`dante` to be installed.

## Installation

### Binary Packages

Recent versions of aerc are available on:
- [Alpine](https://pkgs.alpinelinux.org/packages?name=aerc)
- [Arch](https://archlinux.org/packages/community/x86_64/aerc/)
- [macOS through Homebrew](https://formulae.brew.sh/formula/aerc)

And likely other platforms.

### From Source

Install the dependencies:

- go (>=1.13)
- [scdoc](https://git.sr.ht/~sircmpwn/scdoc)

Then compile aerc:

    $ make

aerc optionally supports notmuch. To enable it, you need to have a recent
version of [notmuch](https://notmuchmail.org/#index7h2), including the header
files (notmuch.h). Then compile aerc with the necessary build tags:

    $ GOFLAGS=-tags=notmuch make

To install aerc locally:

    # make install

## Resources

Send patches and questions to
[~sircmpwn/aerc@lists.sr.ht](https://lists.sr.ht/~sircmpwn/aerc)

Instructions for preparing a patch are available at
[git-send-email.io](https://git-send-email.io)

Subscribe to release announcements on
[~sircmpwn/aerc-announce](https://lists.sr.ht/~sircmpwn/aerc-announce)

Bugs & todo here: [~sircmpwn/aerc2](https://todo.sr.ht/~sircmpwn/aerc2)
