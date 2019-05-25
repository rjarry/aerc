# aerc

aerc is an email client for your terminal.

Join the IRC channel: [#aerc on irc.freenode.net](http://webchat.freenode.net/?channels=aerc&uio=d4)

## Building

Install the dependencies:

- go (compile-time)
- scdoc (compile-time)
- libvterm (compile & runtime)

Then compile aerc:

    $ make

## Installation

    # make install
    $ aerc

On its first run, aerc will copy the default config files to `~/.config/aerc`
and show the account configuration wizard.

## Contributing

[Send patches](https://git-send-email.io) to
[~sircmpwn/aerc@lists.sr.ht](mailto:~sircmpwn/aerc@lists.sr.ht).

Bugs & todo here: [~sircmpwn/aerc2](https://todo.sr.ht/~sircmpwn/aerc2)
