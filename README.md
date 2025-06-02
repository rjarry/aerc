# aerc

[![builds.sr.ht status](https://builds.sr.ht/~rjarry/aerc.svg)](https://builds.sr.ht/~rjarry/aerc)
[![GitHub macOS CI status](https://github.com/rjarry/aerc/actions/workflows/macos.yml/badge.svg)](https://github.com/rjarry/aerc/actions/workflows/macos.yml)

[aerc](https://sr.ht/~rjarry/aerc/) is an email client for your terminal.

This is a fork of [the original aerc](https://git.sr.ht/~sircmpwn/aerc)
by Drew DeVault.

A short demonstration can be found on [https://aerc-mail.org/](https://aerc-mail.org/)

Join the IRC channel: [#aerc on irc.libera.chat](http://web.libera.chat/?channels=aerc)
for end-user support, and development.

## Usage

On its first run, aerc will copy the default config files to `~/.config/aerc`
on Linux or `~/Library/Preferences/aerc` on MacOS (or `$XDG_CONFIG_HOME/aerc` if set)
and show the account configuration wizard.

If you redirect stdout to a file, logging output will be written to that file:

    $ aerc > log

Note that the default HTML filter additionally needs `w3m` to be installed
along with optional `unshare` (from `util-linux`) or `socksify` (from
`dante-utils`).

### Documentation

Also available as man pages:

- [aerc(1)](https://git.sr.ht/~rjarry/aerc/tree/master/item/doc/aerc.1.scd)
- [aerc-accounts(5)](https://git.sr.ht/~rjarry/aerc/tree/master/item/doc/aerc-accounts.5.scd)
- [aerc-binds(5)](https://git.sr.ht/~rjarry/aerc/tree/master/item/doc/aerc-binds.5.scd)
- [aerc-config(5)](https://git.sr.ht/~rjarry/aerc/tree/master/item/doc/aerc-config.5.scd)
- [aerc-imap(5)](https://git.sr.ht/~rjarry/aerc/tree/master/item/doc/aerc-imap.5.scd)
- [aerc-jmap(5)](https://git.sr.ht/~rjarry/aerc/tree/master/item/doc/aerc-jmap.5.scd)
- [aerc-maildir(5)](https://git.sr.ht/~rjarry/aerc/tree/master/item/doc/aerc-maildir.5.scd)
- [aerc-notmuch(5)](https://git.sr.ht/~rjarry/aerc/tree/master/item/doc/aerc-notmuch.5.scd)
- [aerc-patch(7)](https://git.sr.ht/~rjarry/aerc/tree/master/item/doc/aerc-patch.7.scd)
- [aerc-search(1)](https://git.sr.ht/~rjarry/aerc/tree/master/item/doc/aerc-search.1.scd)
- [aerc-sendmail(5)](https://git.sr.ht/~rjarry/aerc/tree/master/item/doc/aerc-sendmail.5.scd)
- [aerc-smtp(5)](https://git.sr.ht/~rjarry/aerc/tree/master/item/doc/aerc-smtp.5.scd)
- [aerc-stylesets(7)](https://git.sr.ht/~rjarry/aerc/tree/master/item/doc/aerc-stylesets.7.scd)
- [aerc-templates(7)](https://git.sr.ht/~rjarry/aerc/tree/master/item/doc/aerc-templates.7.scd)
- [aerc-tutorial(7)](https://git.sr.ht/~rjarry/aerc/tree/master/item/doc/aerc-tutorial.7.scd)
- [carddav-query(1)](https://git.sr.ht/~rjarry/aerc/tree/master/item/doc/carddav-query.1.scd)

User contributions and integration with external tools:

- [wiki](https://man.sr.ht/~rjarry/aerc/)

## Installation

### Binary Packages

Recent versions of aerc are available on:

- [Alpine](https://pkgs.alpinelinux.org/packages?name=aerc)
- [Arch](https://archlinux.org/packages/extra/x86_64/aerc/)
- [Debian](https://tracker.debian.org/pkg/aerc)
- [Fedora](https://packages.fedoraproject.org/pkgs/aerc/aerc/)
- [openSUSE](https://build.opensuse.org/package/show/openSUSE:Factory/aerc)
- [macOS through Homebrew](https://formulae.brew.sh/formula/aerc)
- [Slackware](https://slackbuilds.org/result/?search=aerc)

And likely other platforms.

### From Source

Install the dependencies:

- go (>=1.23) *(Go versions are supported until their end-of-life; support for
  older versions may be dropped at any time due to incompatibilities or newer
  required language features.)*
- [scdoc](https://git.sr.ht/~sircmpwn/scdoc)
- GNU make

Then compile aerc:

    $ gmake

aerc optionally supports notmuch. To enable it, you need to have a recent
version of [notmuch](https://notmuchmail.org/#index7h2), including the header
files (notmuch.h). The `notmuch` build tag should be automatically added. To
check if it is, run the following command:

    $ ./aerc -v
    aerc 0.14.0-108-g31e1cd9af565 +notmuch (go1.19.6 amd64 linux)
                                  ^^^^^^^^

If it is not, you can force it before building:

    $ gmake GOFLAGS=-tags=notmuch

If you have notmuch headers available but do not want to build notmuch support
in aerc, force GOFLAGS to an empty value:

    $ gmake GOFLAGS=

To install aerc locally:

    # gmake install

By default, aerc will install config files to directories under `/usr/local/aerc`,
and will search for templates and stylesets in these locations in order:

- `${XDG_CONFIG_HOME:-~/.config}/aerc`
- `${XDG_DATA_HOME:-~/.local/share}/aerc`
- `/usr/local/share/aerc`
- `/usr/share/aerc`

At build time it is possible to add an extra location to this list and to use
that location as the default install location for config files by setting the
`PREFIX` option like so:

    # gmake PREFIX=/custom/location
    # gmake install PREFIX=/custom/location

This will install templates and other config files to `/custom/location/share/aerc`,
and man pages to `/custom/location/share/man`. This extra location will have lower
priority than the XDG locations but higher than the fixed paths.

## Contributing

Anyone can contribute to aerc. Please refer to [the contribution
guidelines](https://git.sr.ht/~rjarry/aerc/tree/master/item/CONTRIBUTING.md)

## Resources

Ask for support or follow general discussions on
[~rjarry/aerc-discuss@lists.sr.ht](https://lists.sr.ht/~rjarry/aerc-discuss).

Send patches and development related questions to
[~rjarry/aerc-devel@lists.sr.ht](https://lists.sr.ht/~rjarry/aerc-devel).

Instructions for preparing a patch are available at
[git-send-email.io](https://git-send-email.io)

Subscribe to release announcements on
[~rjarry/aerc-announce@lists.sr.ht](https://lists.sr.ht/~rjarry/aerc-announce)

Submit *confirmed* bug reports and *confirmed* feature requests on
[https://todo.sr.ht/~rjarry/aerc](https://todo.sr.ht/~rjarry/aerc).

Send security vulnerability reports via emails to
[~rjarry/aerc-security@lists.sr.ht](~rjarry/aerc-security@lists.sr.ht)
encrypted with the GPG key located at
[`contrib/signing-key.asc`](https://git.sr.ht/~rjarry/aerc/tree/master/item/contrib/signing-key.asc)
(fingerprint `DC07 18E3 22E2 C760 5EBD  C831 4695 7EC0 8FD0 FE90`).

[License](https://git.sr.ht/~rjarry/aerc/tree/master/item/LICENSE).

[Change log](https://git.sr.ht/~rjarry/aerc/tree/master/item/CHANGELOG.md).
