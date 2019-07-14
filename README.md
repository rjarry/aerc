# aerc

aerc is an email client for your terminal.

Join the IRC channel: [#aerc on irc.freenode.net](http://webchat.freenode.net/?channels=aerc&uio=d4)
for end-user support, and [#aerc-dev](http://webchat.freenode.net/?channels=aerc-dev&uio=d4)
for development.

## Building

Install the dependencies:

- go (>=1.12)
- scdoc

Then compile aerc:

    $ make

## Installation

    # make install
    $ aerc

On its first run, aerc will copy the default config files to `~/.config/aerc`
and show the account configuration wizard.

If you redirect stdout to a file, logging output will be written to that file:

    $ aerc > log

## Resources

[Send patches](https://git-send-email.io) and questions to
[~sircmpwn/aerc@lists.sr.ht](https://lists.sr.ht/~sircmpwn/aerc).

Subscribe to release announcements on
[~sircmpwn/aerc-announce](https://lists.sr.ht/~sircmpwn/aerc-announce)

Bugs & todo here: [~sircmpwn/aerc2](https://todo.sr.ht/~sircmpwn/aerc2)
