# aerc

aerc is a *work in progress* email client for your terminal.

It is not yet suitable for daily use.

Join the IRC channel: [#aerc on irc.freenode.net](http://webchat.freenode.net/?channels=aerc&uio=d4)

## Building

aerc depends on:

- go (compile-time)
- scdoc (compile-time)
- libvterm (compile & runtime)

    $ make

## Installation

    # make install
    $ man aerc

## Usage

```
$ mkdir ~/.config/aerc
$ cp config/*.conf ~/.config/aerc/
$ vim ~/.config/aerc/accounts.conf
```

Fill in your account details and configure the rest to taste, then run `aerc`.

## Contributing

Send patches to
[~sircmpwn/aerc@lists.sr.ht](mailto:~sircmpwn/aerc@lists.sr.ht).

Bugs & todo here: [~sircmpwn/aerc2](https://todo.sr.ht/~sircmpwn/aerc2)
