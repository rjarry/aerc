# Change Log

All notable changes to aerc will be documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased](https://git.sr.ht/~rjarry/aerc/log/master)

### Added

- Add a `-A` option to `:save` for saving all the named parts, not just
  attachments
- Add `<Backspace>` key to bindings
- Colorize can style diff chunk function names with `diff_chunk_func`.
- Warn before sending emails with an empty subject with `empty-subject-warning`
  in `aerc.conf`.
- IMAP now uses the delimiter advertised by the server
- Completions for `:mkdir`
- `carddav-query` utility to use for `address-book-cmd`.
- JMAP support.
- Folder name mapping with `folder-map` in `accounts.conf`.
- Add option `-d` to `:open` to automatically delete temporary files.
- Edit email headers directly in the text editor with `[compose].edit-headers`
  in `aerc.conf` or with the `-e` flag for all compose related commands (e.g.
  `:compose`, `:forward`, `:recall`, etc.).

### Fixed

- `:archive` now works on servers using a different delimiter
- `:save -a` now works with multiple attachments with the same filename
- `:open` uses the attachment extension for temporary files, if possible

### Changed

- Names formatted like "Last Name, First Name" are better supported in templates
- Composing an email is now aborted if the text editor exits with an error
  (e.g. with `vim`, abort an email with `:cq`).

## [0.15.2](https://git.sr.ht/~rjarry/aerc/refs/0.15.2) - 2023-05-11

### Fixed

- Extra messages disappearing when deleting on maildir.
- `colorize` and `wrap` filters option parsing on ARM.

## [0.15.1](https://git.sr.ht/~rjarry/aerc/refs/0.15.1) - 2023-04-28

### Fixed

- Embedded terminal partial refreshes.
- Maildir message updates after `mbsync`.

## [0.15.0](https://git.sr.ht/~rjarry/aerc/refs/0.15.0) - 2023-04-26

### Added

- New column-based message list format with `index-columns`.
- Add a `msglist_answered` style for answered messages.
- Compose `Format=Flowed` messages with `format-flowed=true` in `aerc.conf`.
- Add a `trimSignature` function to the templating engine.
- Change local domain name for SMTP with `smtp-domain=example.com` in
  `aerc.conf`
- New column-based status line format with `status-columns`.
- Inline user-defined styles can be inserted in UI templates via the
  `{{.Style "name" string}}` function.
- Add the ability to run arbitrary commands over the socket. This can be
  disabled using the `disable-ipc` setting.
- Allow configuring URL handlers via `x-scheme-handler/<scheme>` `[openers]` in
  `aerc.conf`.
- Allow basic shell globbing in `[openers]` MIME types.
- Dynamic `msglist_*` styling based on email header values in stylesets.
- Add `mail-received`, `aerc-startup`, and `aerc-shutdown` hooks.
- Search/filter by flags with the `-H` flag.

### Changed

- Filters are now installed in `$PREFIX/libexec/aerc/filters`. The default exec
  `PATH` has been modified to include all variations of the `libexec` subdirs.
- The built-in `colorize` filter theme is now configured in styleset files into
  the `[viewer]` section.
- The standard Usenet signature delimiter `"-- "` is now prepended to
  `signature-file` and `signature-cmd` if not already present.
- All `aerc(1)` commands now interpret `aerc-templates(7)` markup.
- running commands (like mailto: or mbox:) no longer prints a success message
- The built-in `colorize` filter now emits OSC 8 to mark URLs and emails. Set
  `[general].enable-osc8 = true` in `aerc.conf` to enable it.
- Notmuch support is now automatically enabled when `notmuch.h` is detected on
  the system.

### Deprecated

- `[ui].index-format` setting has been replaced by `index-columns`.
- `[statusline].render-format` has been replaced by `status-columns`.
- Removed support for go < 1.18.
- Removed support for `[ui:subject...]` contextual sections in `aerc.conf`.
- `[triggers]` setting has been replaced by `[hooks]`.
- `smtp-starttls` setting in `accounts.conf` has been removed. All `smtp://`
  transports now assume `STARTTLS` and will fail if the server does not support
  it. To disable `STARTTLS`, use `smtp+insecure://`.

## [0.14.0](https://git.sr.ht/~rjarry/aerc/refs/0.14.0) - 2023-01-04

### Added

- View common email envelope headers with `:envelope`.
- Notmuch accounts now support maildir operations: `:copy`, `:move`, `:mkdir`,
  `:rmdir`, `:archive` and the `copy-to` option.
- Display messages from bottom to top with `[ui].reverse-msglist-order=true` in
  `aerc.conf`.
- Display threads from bottom to top with `[ui].reverse-thread-order=true` in
  `aerc.conf`.
- Style search results in the message list with `msglist_result.*`.
- Preview messages with their attachments before sending with `:preview`.
- Filter commands now have `AERC_FORMAT`, `AERC_SUBJECT` and `AERC_FROM`
  defined in their environment.
- Override the subject prefix for replies pattern with `subject-re-pattern` in
  `accounts.conf`.
- Search/filter by absolute and relative date ranges with the `-d` flag.
- LIST-STATUS and ORDEREDSUBJECT threading extensions support for imap.
- Built-in `wrap` filter that does not mess up nested quotes and lists.
- Write `multipart/alternative` messages with `:multipart` and commands defined
  in the new `[multipart-converters]` section of `aerc.conf`.
- Close the message viewer before opening the composer with `:reply -c`.
- Attachment indicator in message list flags (by default `a`, but can be
  changed via `[ui].icon-attachment` in `aerc.conf`).
- Open file picker menu with `:attach -m`. The menu must be generated by an
  external command configured via `[compose].file-picker-cmd` in `aerc.conf`.
- Sample stylesets are now installed in `$PREFIX/share/aerc/stylesets`.
- The built-in `colorize` filter now has different themes.

### Changed

- `pgp-provider` now defaults to `auto`. It will use the system `gpg` unless
  the internal keyring exists and contains at least one key.
- Calling `:split` or `:vsplit` without specifying a size, now attempts to use
  the terminal size to determine a useful split-size.

### Fixed

- `:pipe -m git am -3` on patch series when `Message-Id` headers have not been
  generated by `git send-email`.
- Overflowing text in header editors while composing can now be scrolled
  horizontally.

### Deprecated

- Removed broken `:set` command.

## [0.13.0](https://git.sr.ht/~rjarry/aerc/refs/0.13.0) - 2022-10-20

### Added

- Support for bindings with the Alt modifier.
- Zoxide support with `:z`.
- Hide local timezone with `send-as-utc = true` in `accounts.conf`.
- Persistent command history in `~/.cache/aerc/history`.
- Cursor shape support in embedded terminals.
- Bracketed paste support.
- Display current directory in `status-line.render-format` with `%p`.
- Change accounts while composing a message with `:switch-account`.
- Override `:open` handler on a per-MIME-type basis in `aerc.conf`.
- Specify opener as the first `:open` param instead of always using default
  handler (i.e. `:open gimp` to open attachment in GIMP).
- Restored XOAUTH2 support for IMAP and SMTP.
- Support for attaching files with `mailto:`-links
- Filter commands now have the `AERC_MIME_TYPE` and `AERC_FILENAME` variables
  defined in their environment.
- Warn before sending emails that may need an attachment with
  `no-attachment-warning` in `aerc.conf`.
- 3 panel view via `:split` and `:vsplit`
- Configure dynamic date format for the message viewer with
  `message-view-this-*-time-format`.
- View message without marking it as seen with `:view -p`.

### Changed

- `:open-link` now supports link types other than HTTP(S)
- Running the same command multiple times only adds one entry to the command
  history.
- Embedded terminal backend (libvterm was replaced by a pure go implementation).
- Filter commands are now executed with
  `:~/.config/aerc/filters:~/.local/share/aerc/filters:$PREFIX/share/aerc/filters:/usr/share/aerc/filters`
  appended to the exec `PATH`. This allows referencing aerc's built-in filter
  scripts from their name only.

### Fixed

- `:open-link` will now detect links containing an exclamation mark
- `outgoing-cred-cmd` will no longer be executed every time an email needs to
  be sent. The output will be stored until aerc is shut down. This behaviour
  can be disabled by setting `outgoing-cred-cmd-cache=false` in
  `accounts.conf`.
- Mouse support for embedded editors when `mouse-enabled=true`.
- Numerous race conditions.

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
