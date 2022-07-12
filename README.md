# aerc

[![builds.sr.ht status](https://builds.sr.ht/~rjarry/aerc.svg)](https://builds.sr.ht/~rjarry/aerc)
[![GitHub macOS CI status](https://github.com/rjarry/aerc/actions/workflows/macos.yml/badge.svg)](https://github.com/rjarry/aerc/actions/workflows/macos.yml)

[aerc](https://sr.ht/~rjarry/aerc/) is an email client for your terminal.

This is a fork of [the original aerc](https://git.sr.ht/~sircmpwn/aerc)
by Drew DeVault.

A short demonstration can be found on [https://aerc-mail.org/](https://aerc-mail.org/)

Join the IRC channel: [#aerc on irc.libera.chat](http://web.libera.chat/?channels=aerc&uio=d4)
for end-user support, and development.

## Usage

On its first run, aerc will copy the default config files to `~/.config/aerc`
on Linux or `~/Library/Preferences/aerc` on MacOS (or `$XDG_CONFIG_HOME/aerc` if set)
and show the account configuration wizard.

If you redirect stdout to a file, logging output will be written to that file:

    $ aerc > log

Note that the example HTML filter (off by default), additionally needs `w3m` and
`dante` to be installed.

### Documentation

Also available as man pages:

- [aerc(1)](https://git.sr.ht/~rjarry/aerc/tree/master/item/doc/aerc.1.scd)
- [aerc-config(5)](https://git.sr.ht/~rjarry/aerc/tree/master/item/doc/aerc-config.5.scd)
- [aerc-imap(5)](https://git.sr.ht/~rjarry/aerc/tree/master/item/doc/aerc-imap.5.scd)
- [aerc-maildir(5)](https://git.sr.ht/~rjarry/aerc/tree/master/item/doc/aerc-maildir.5.scd)
- [aerc-notmuch(5)](https://git.sr.ht/~rjarry/aerc/tree/master/item/doc/aerc-notmuch.5.scd)
- [aerc-search(1)](https://git.sr.ht/~rjarry/aerc/tree/master/item/doc/aerc-search.1.scd)
- [aerc-sendmail(5)](https://git.sr.ht/~rjarry/aerc/tree/master/item/doc/aerc-sendmail.5.scd)
- [aerc-smtp(5)](https://git.sr.ht/~rjarry/aerc/tree/master/item/doc/aerc-smtp.5.scd)
- [aerc-stylesets(7)](https://git.sr.ht/~rjarry/aerc/tree/master/item/doc/aerc-stylesets.7.scd)
- [aerc-templates(7)](https://git.sr.ht/~rjarry/aerc/tree/master/item/doc/aerc-templates.7.scd)
- [aerc-tutorial(7)](https://git.sr.ht/~rjarry/aerc/tree/master/item/doc/aerc-tutorial.7.scd)

User contributions and integration with external tools:

- [wiki](https://man.sr.ht/~rjarry/aerc/)

## Installation

### Binary Packages

Recent versions of aerc are available on:

- [Alpine](https://pkgs.alpinelinux.org/packages?name=aerc)
- [Arch](https://archlinux.org/packages/community/x86_64/aerc/)
- [Debian](https://tracker.debian.org/pkg/aerc)
- [Fedora](https://packages.fedoraproject.org/pkgs/aerc/aerc/)
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

By default, aerc will install config files to directories under `/usr/local/aerc`,
and will search for templates and stylesets in these locations in order:

- `${XDG_CONFIG_HOME:-~/.config}/aerc`
- `${XDG_DATA_HOME:-~/.local/share}/aerc`
- `/usr/local/share/aerc`
- `/usr/share/aerc`

At build time it is possible to add an extra location to this list and to use
that location as the default install location for config files by setting the
`PREFIX` option like so:

    # make PREFIX=/custom/location
    # make install PREFIX=/custom/location

This will install templates and other config files to `/custom/location/share/aerc`,
and man pages to `/custom/location/share/man`. This extra location will have lower
priority than the XDG locations but higher than the fixed paths.

## Contribution Quick Start

Anyone can contribute to aerc. First you need to clone the repository and build
the project:

    $ git clone https://git.sr.ht/~rjarry/aerc
    $ cd aerc
    $ make

Patch the code. Make some tests. Ensure that your code is properly formatted
with gofmt. Ensure that everything builds and works as expected. Ensure that
you did not break anything.

- If applicable, update unit tests.
- If adding a new feature, please consider adding new tests.
- Do not forget to update the docs.
- If your commit brings visible changes for end-users, add an entry in the
  *Unreleased* section of the
  [CHANGELOG.md](https://git.sr.ht/~rjarry/aerc/tree/master/item/CHANGELOG.md)
  file.

Once you are happy with your work, you can create a commit (or several
commits). Follow these general rules:

- Limit the first line (title) of the commit message to 60 characters.
- Use a short prefix for the commit title for readability with `git log --oneline`.
- Use the body of the commit message to actually explain what your patch does
  and why it is useful.
- Address only one issue/topic per commit.
- If you are fixing a ticket, use appropriate
  [commit trailers](https://man.sr.ht/git.sr.ht/#referencing-tickets-in-git-commit-messages).
- If you are fixing a regression introduced by another commit, add a `Fixes:`
  trailer with the commit id and its title.

There is a great reference for commit messages in the
[Linux kernel documentation](https://www.kernel.org/doc/html/latest/process/submitting-patches.html#describe-your-changes).

IMPORTANT: you must sign-off your work using `git commit --signoff`. Follow the
[Linux kernel developer's certificate of origin][linux-signoff] for more
details. All contributions are made under the MIT license. If you do not want
to disclose your real name, you may sign-off using a pseudonym. Here is an
example:

    Signed-off-by: Robin Jarry <robin@jarry.cc>

[linux-signoff]: https://www.kernel.org/doc/html/latest/process/submitting-patches.html#sign-your-work-the-developer-s-certificate-of-origin

Before sending the patch, you should configure your local clone with sane
defaults:

    $ git config format.subjectPrefix "PATCH aerc"
    $ git config sendemail.to "~rjarry/aerc-devel@lists.sr.ht"

And send the patch to the mailing list:

    $ git send-email --annotate -1

Before your patch can be applied, it needs to be reviewed and approved by
others. They will indicate their approval by replying to your patch with
a [Tested-by, Reviewed-by or Acked-by][linux-review] trailer. For example:

    Acked-by: Robin Jarry <robin@jarry.cc>

[linux-review]: https://www.kernel.org/doc/html/latest/process/submitting-patches.html#using-reported-by-tested-by-reviewed-by-suggested-by-and-fixes

There is no "chain of command" in aerc. Anyone that feels comfortable enough to
"ack" or "review" a patch should express their opinion freely with an official
Acked-by or Reviewed-by trailer. If you only tested that a patch works as
expected but did not conduct a proper code review, you can indicate it with
a Tested-by trailer.

You can follow the review process via email and on the
[web ui](https://lists.sr.ht/~rjarry/aerc-devel/patches).

Wait for feedback. Address comments and amend changes to your original commit.
Then you should send a v2 (and maybe a v3, v4, etc.):

    $ git send-email --annotate -v2 -1

Be polite, patient and address *all* of the reviewers' remarks. If you disagree
with something, feel free to discuss it.

Once your patch has been reviewed and approved (and if the maintainer is OK
with it), it will be applied and pushed.

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

[License](https://git.sr.ht/~rjarry/aerc/tree/master/item/LICENSE).

[Change log](https://git.sr.ht/~rjarry/aerc/tree/master/item/CHANGELOG.md).
