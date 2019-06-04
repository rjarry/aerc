# aerc

aerc is an email client for your terminal.

Join the IRC channel: [#aerc on irc.freenode.net](http://webchat.freenode.net/?channels=aerc&uio=d4)

## Building

Install the dependencies:

- go (>=1.12)
- scdoc

aerc optionally depends on the following for display filters (you'll have to
change the default aerc.conf if you don't want these):

- python (>=3.7)
- colorama
- w3m
- sockify

Then compile aerc:

    $ make

## Installation

    # make install
    $ aerc

On its first run, aerc will copy the default config files to `~/.config/aerc`
and show the account configuration wizard.

## Resources

[Send patches](https://git-send-email.io) and questions to
[~sircmpwn/aerc@lists.sr.ht](https://lists.sr.ht/~sircmpwn/aerc).

Subscribe to release announcements on
[~sircmpwn/aerc-announce](https://lists.sr.ht/~sircmpwn/aerc-announce)

Bugs & todo here: [~sircmpwn/aerc2](https://todo.sr.ht/~sircmpwn/aerc2)
