# Change Log

All notable changes to aerc will be documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased](https://git.sr.ht/~rjarry/aerc/log/master)

### Added

- Support for bindings with the Alt modifier.

### Fixed

- `outgoing-cred-cmd` will no longer be executed every time an email needs to
  be sent. The output will be stored until aerc is shut down. This behaviour
  can be disabled by setting `outgoing-cred-cmd-cache=false` in
  `accounts.conf`.

## [0.12.0](https://git.sr.ht/~rjarry/aerc/refs/0.12.0) - 2022-09-01

### Added

- Read-only mbox backend support.
- Import/Export mbox files with `:import-mbox` and `:export-mbox`.
- `address-book-cmd` can now also be specified in `accounts.conf`.
- Run `check-mail-cmd` with `:check-mail`.
- Display active key binds with `:help keys` (bound to `?` by default).
- Multiple visual selections with `:mark -V`.
- Mark all messages of the same thread with `:mark -T`.
- Set default collapse depth of directory tree with `dirlist-collapse`.

### Changed

- Aerc will no longer exit while a send is in progress.
- When scrolling through large folders, client side threading is now debounced
  to avoid lagging. This can be configured with `client-threads-delay`.
- The provided awk filters are now POSIX compliant and should work on MacOS and
  BSD.
- `outgoing-cred-cmd` execution is now deferred until a message needs to be sent.
- `next-message-on-delete` now also applies to `:archive`.
- `:attach` now supports path globbing (`:attach *.log`)

### Fixed

- Transient crashes when closing tabs.
- Binding a command to `<c-i>` and `<c-m>`.
- Reselection after delete and scroll when client side threading is enabled.
- Background mail count polling when the default folder is empty on startup.
- Wide character handling in the message list.
- Issues with message reselection during scrolling and after `:delete` with
  threading enabled.

### Deprecated

- Removed support for go < 1.16.

## [0.11.0](https://git.sr.ht/~rjarry/aerc/refs/0.11.0) - 2022-07-11

### Added

- Deal with calendar invites with `:accept`, `:accept-tentative` and `:decline`.
- IMAP cache support.
- Maildir++ support.
- Background mail count polling for all folders.
- Authentication-Results display (DKIM, SPF & DMARC).
- Folder-specific key bindings.
- Customizable PGP icons.
- Open URLs from messages with `:open-link`.
- Forward all individual attachments with `:forward -A`.

### Changed

- Messages are now deselected after performing a command. Use `:remark` to
  reselect the previously selected messages and chain other commands.
- Pressing `<Enter>` in the default postpone folder now runs `:recall` instead
  of `:view`.
- PGP signed/encrypted indicators have been reworked.
- The `threading-enabled` option now affects if message threading should be
  enabled at startup. This option no longer conflicts with `:toggle-threads`.

### Fixed

- `:pipe`, `:save` and `:open` for signed and/or encrypted PGP messages.
- Messages that have failed `gpg` encryption/signing are no longer sent.
- Recalling attachments from drafts.

## [0.10.0](https://git.sr.ht/~rjarry/aerc/refs/0.10.0) - 2022-05-07

### Added

- Format specifier for compact folder names in dirlist.
- Customizable, per-folder status line.
- Allow binding commands to `<` and `>` keys.
- Optional filter to parse ICS files (uses `python3` vobject library).
- Save all attachments with `:save -a`.
- Native `gpg` support.
- PGP `auto-sign` and `opportunistic-encrypt` options.
- Attach your PGP public key to a message with `:attach-key`.

### Fixed

- Stack overflow with faulty `References` headers when `:toggle-threads` is
  enabled.

## [0.9.0](https://git.sr.ht/~rjarry/aerc/refs/0.9.0) - 2022-03-21

### Added

- Allow `:pipe` on multiple selected messages.
- Client side on-the-fly message threading with `:toggle-threads` (conflicts
  with existing `threading-enabled` option).
- Per-account, better status line.
- Consecutive, incremental `:search` and `:filter` support.
- Foldable tree for directory list.
- `Bcc` and `Body` in `mailto:` handler.
- Fuzzy tab completion for commands and folders.
- Key pass though mode for the message viewer to allow searching with `less`.

### Changed

- Use terminfo for setting terminal title.

## [0.8.2](https://git.sr.ht/~rjarry/aerc/refs/0.8.2) - 2022-02-19

### Added

- New `colorize` filter with diff, multi-level quotes and URL coloring.
- XDG desktop entry to use as default `mailto:` handler.
- IMAP automatic reconnect.
- Recover drafts after crash with `:recover`.
- Show possible actions with user configured bindings when reviewing a message.
- Allow setting any header in email templates.
- Improved `:change-folder` responsiveness.
- New `:compose` option to never include your own address when replying.

### Changed

- Templates and style sets are now searched from multiple directories. Not from
  a single hard-coded folder set at build time. In addition of the configured
  `PREFIX/share/aerc` folders at build time, aerc now also looks into
  `~/.config/aerc`, `~/.local/share/aerc`, `/usr/local/share/aerc` and
  `/usr/share/aerc`
- A warning is displayed when trying to configure account specific bindings
  for a non-existent account.

### Fixed

- `Ctrl-h` binding not working.
- Open files leaks for maildir and notmuch.

## 0.8.1 - 2022-02-20 [YANKED]

## 0.8.0 - 2022-02-19 [YANKED]

## [0.7.1](https://git.sr.ht/~rjarry/aerc/refs/0.7.1) - 2022-01-15

### Added

- IMAP low level TCP settings.
- Experimental IMAP server-side and notmuch threading.
- `:recall` now works from any folder.
- PGP/MIME signing and encryption.
- Account specific bindings.

### Fixed

- Address book completion for multiple addresses.
- Maildir external mailbox changes monitoring.

## 0.7.0 - 2022-01-14 [YANKED]

## [0.6.0](https://git.sr.ht/~rjarry/aerc/refs/0.6.0) - 2021-11-09

*The project was forked to <https://git.sr.ht/~rjarry/aerc>.*

### Added

- Allow more modifiers for key bindings.
- Dynamic dates in message list.
- Match any header in filters specifiers.

### Fixed

- Don't read entire messages into memory.

## [0.5.0](https://git.sr.ht/~sircmpwn/aerc/refs/0.5.0) - 2020-11-10

### Added

- Remove folder with `:rmdir`.
- Configurable style sets.
- UI context aware options and styling.
- oauthbearer support for SMTP.
- IMAP sort support.

## [0.4.0](https://git.sr.ht/~sircmpwn/aerc/refs/0.4.0) - 2020-05-20

### Added

- Address book completion.
- Initial PGP support using an internal key store.
- Messages can now be selected with `:mark`.
- Drafts handing with `:postpone` and `:recall`.
- Tab management with `:move-tab` and `:pin-tab`.
- Add arbitrary headers in the compose window with `:header`.
- Interactive prompt with `:choose`.
- Notmuch labels improvements.
- Support setting some headers in message templates.

### Changed

- `aerc.conf` ini parser only uses `=` as delimiter. `:` is now ignored.

## [0.3.0](https://git.sr.ht/~sircmpwn/aerc/refs/0.3.0) - 2019-11-21

### Added

- A new notmuch backend is available. See `aerc-notmuch(5)` for details.
- Message templates now let you change the default reply and forwarded message
  templates, as well as add new templates of your own. See `aerc-templates(7)`
  for details.
- Mouse input is now optionally available and has been rigged up throughout the
  UI, set `[ui]mouse-enabled=true` in `aerc.conf` to enable.
- `:cc` and `:bcc` commands are available in the message composer.
- Users may now configure arbitrary message headers for editing in the message
  composer.

## [0.2.0](https://git.sr.ht/~sircmpwn/aerc/refs/0.2.0) - 2019-07-29

### Added

- Maildir & sendmail transport support
- Search and filtering are supported (via `/` and `\` by default)
- `aerc mailto:...` now opens the composer in running aerc instance
- Initial tab completion support has been added
- Improved headers and addressing in the composer and message view
- Message attachments may now be added in the composer
- Commands can now be run in the background with `:exec` or `:pipe -b`
- A new triggers system allows running aerc commands when new emails arrive,
  which may (for example) be used to send desktop notifications or move new
  emails to a folder

### Changed

- The filters have been rewritten in awk, dropping the Python dependencies.
  `w3m` and `dante` are both still required for HTML email, but the HTML filter
  has been commented out in the default config file.
- The default keybindings and configuration options have changed considerably,
  and users are encouraged to pull the latest versions out of `/usr/share` and
  re-apply their modifications to them, or to at least review the diff with
  their current configurations. aerc may not behave properly without taking
  this into account.

## [0.1.0](https://git.sr.ht/~sircmpwn/aerc/refs/0.1.0) - 2019-06-03

Initial release.
