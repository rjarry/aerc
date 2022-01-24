# aerc

[![builds.sr.ht status](https://builds.sr.ht/~rjarry/aerc.svg)](https://builds.sr.ht/~rjarry/aerc)

[aerc](https://sr.ht/~rjarry/aerc/) is an email client for your terminal.

This is a fork of [the original aerc](https://git.sr.ht/~sircmpwn/aerc)
by Drew DeVault.

A short demonstration can be found on [https://aerc-mail.org/](https://aerc-mail.org/)

Join the IRC channel: [#aerc on irc.libera.chat](http://web.libera.chat/?channels=aerc&uio=d4)
for end-user support, and [#aerc-dev](http://web.libera.chat/?channels=aerc-dev&uio=d4)
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

Before sending the patch, you should configure your local clone with sane
defaults:

    $ git config format.subjectPrefix "PATCH aerc"
    $ git config sendemail.to "~rjarry/aerc-devel@lists.sr.ht"

And send the patch to the mailing list:

    $ git send-email --annotate -1

Wait for feedback. Address comments and amend changes to your original commit.
Then you should send a v2:

    $ git send-email --in-reply-to=$first_message_id --annotate -v2 -1

Once the maintainer is happy with your patch, they will apply it and push it.

## Resources

Send patches and questions to
[~rjarry/aerc-devel@lists.sr.ht](https://lists.sr.ht/~rjarry/aerc-devel).

Instructions for preparing a patch are available at
[git-send-email.io](https://git-send-email.io)

Subscribe to release announcements on
[~rjarry/aerc-announce@lists.sr.ht](https://lists.sr.ht/~rjarry/aerc-announce)

Submit bug reports and feature requests on
[https://todo.sr.ht/~rjarry/aerc](https://todo.sr.ht/~rjarry/aerc).
