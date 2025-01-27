# Change Log

All notable changes to aerc will be documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [0.20.1](https://git.sr.ht/~rjarry/aerc/refs/0.20.1) - 2025-01-27

### Fixed

- `:sign` and `:encrypt` descriptions overflow the review screen.
- Some messages are hidden when using client side threading.

## [0.20.0](https://git.sr.ht/~rjarry/aerc/refs/0.20.0) - 2025-01-25

### Added

- `copy-to` now supports multiple destination folders.
- All commands that involve composing messages (`:compose`, `:reply`,
  `:recall`, `:unsubscribe` and `:forward`) now have a new `-s` flag to skip
  opening the text editor and go directly to the review screen. Previously,
  this flag was restricted to calendar invitations response commands
  (`:accept`, `:accept-tentative` and `:decline`).

### Fixed

- `copy-to-replied` now properly works without having `copy-to` also set.
- `copy-to-replied` creates empty messages when `copy-to` is also set.
- The address-book completion popovers now again appear under the field being
  completed.
- The new-message bell is now rung again for threaded directories as well.

### Changed

- The `default` styleset status line background has been reset to the default
  color (light or dark, depending on your terminal color scheme) in order to
  make error, warning or success messages more readable.
- Key bindings in the compose review screen are now displayed in the order in
  which they are defined in the `[compose::review]` section of `binds.conf`.
- It is now possible to explicitly hide key bindings from the compose review
  screen by using a special ` # -` annotation.

### Closed Tickets

- [#296: :compose: add flag to go directly to review screen](https://todo.sr.ht/~rjarry/aerc/296)

## [0.19.0](https://git.sr.ht/~rjarry/aerc/refs/0.19.0) - 2025-01-14

### Added

- New `:redraw` command to force a repaint of the screen.
- New `head` and `tail` templates functions for strings.
- New `{{.AccountFrom}}` template variable.
- Replying to all will include the Sender in Cc.
- Add `-b` flag to the `:view` command to open messages in a background tab.
- `AERC_ACCOUNT` and `AERC_FOLDER` are now available in the signature command
  environment.
- Filters will receive the actual `COLUMNS` and `LINES` values.
- The `:forward` command now sets the forwarded flag.
- Forwarded messages can now be searched for and filtered in notmuch and
  maildir.
- Forwarded messages can be styled differently in the message list.
- Forwarded messages can be identified with the `{{.IsForwarded}}` template.
- The `:flag` command now sets/unsets/toggle the forwarded tag.
- The notmuch backend now honors the forwarded flag, setting the `passed` tag.
- The maildir backend now honors the `forwarded`/`passed` flag.
- Auto-switch projects based on the message subject for the `:patch` command.
- New `:echo` command that prints its arguments with templates resolved.
- New `use-envelope-from` option in `accounts.conf`.
- Command completion now displays descriptions next to completion items.
- New `completion_description` style object in style sets used for rendering
  completion item descriptions.
- `:import-mbox` can now import data from an URL.
- Dynamic message list style can now match on multiple email headers.
- The JMAP backend now supports full thread fetching and caching (limited
  within a single mailbox).
- `:expand-folder` and `:collapse-folder` can now act on a non selected folder.
- Filters commands can now provide their own paging by prefixing them with a
  `!` character. Doing so will disable the configured `[viewer].pager` and
  connect them directly to the terminal.
- Reply to addresses in `From` and `Reply-To` headers with `:reply -f`.

### Fixed

- Builtin `calendar` filter shows empty attendee list.
- Terminal-based pinentry programs (e.g. `pinentry-curses`) now work properly.
- Failure to create IPC socket on Gentoo.
- Notmuch searches which explicitly contain tags from `exclude-tags` now return
  messages.
- Invitations now honor the `:send -a` flag.
- Remove unwanted less than symbol from In-Reply-To header when Message-ID uses
  folding.
- Aliases are now taken into account correctly when replying to own messages
  such as from the Sent folder or via a mailing list.
- Some SMTP servers do not strip `Bcc` headers. aerc now removes them before
  sending emails to avoid leaking private information. A new `strip-bcc =
  false` option can be used in `accounts.conf` to revert to previous behaviour
  (preserve `Bcc` headers in outgoing messages).
- There should no longer be any duplicates in recipient lists when replying.
- GPG signatures and encrypted parts now use CRLF line endings as required by
  RFC 5322.

### Changed

- Template function `quote` only prefixes with a space if at quote depth `1`.
- Templates passed to the `:reply` command using the `-T` flag can now make use
  of `{{.OriginalText}}`.
- The location of the command history file has changed to
  `${XDG_STATE_HOME:-$HOME/.local/state}/aerc/history`.
- Tab completions for text fields are run asynchronously. In-flight requests
  are cancelled when new input arrives.
- Path completion now uses the normal filtering mechanism, respecting case
  sensitivity and the fuzzy completion option.
- The `html` filter is now enabled by default, making `w3m` a weak runtime
  dependency. If it is not installed, viewing HTML emails will fail with an
  explicit error.
- The default `text/html` filter will now run `w3m` in interactive mode.
- The builtin `html` and `html-unsafe` filters can now take additional
  arguments that will be passed to `w3m`. This can be used to enable inline
  images when viewing `text/html` parts (e.g.: `text/html = ! html-unsafe
  -sixel`).
- The templates `exec` commands is now executed with the `filters` exec `$PATH`
  similar to filter commands.
- The default `quoted_reply` template now converts `text/html` parts to plain
  text before quoting them.

### Deprecated

- Support for go 1.20 and older.

### Closed Tickets

- [#150: Expand .Account with .Address and .Name](https://todo.sr.ht/~rjarry/aerc/150)
- [#202: pinentry-tty breaks aerc](https://todo.sr.ht/~rjarry/aerc/202)
- [#215: regression since bumping go-maildir](https://todo.sr.ht/~rjarry/aerc/215)
- [#220: add trim to templates](https://todo.sr.ht/~rjarry/aerc/220)
- [#226: Automatic patch switch based on email prefix](https://todo.sr.ht/~rjarry/aerc/226)
- [#232: $(tput cols) in filters report 80 all the time](https://todo.sr.ht/~rjarry/aerc/232)
- [#238: Implement decryption on action {cp,mv,pipe}](https://todo.sr.ht/~rjarry/aerc/238)
- [#250: allow disabling pager in filter](https://todo.sr.ht/~rjarry/aerc/250)
- [#259: :reply -a should reply to Sender as well](https://todo.sr.ht/~rjarry/aerc/259)
- [#266: Add opening individual emails in the background](https://todo.sr.ht/~rjarry/aerc/266)
- [#271: Add documentation to options in the autocomplete menu](https://todo.sr.ht/~rjarry/aerc/271)
- [#277: add :echo command](https://todo.sr.ht/~rjarry/aerc/277)
- [#281: Unable to open local `mbox` files](https://todo.sr.ht/~rjarry/aerc/281)
- [#283: BCC headers are exposed to recipients with gmail](https://todo.sr.ht/~rjarry/aerc/283)
- [#287: Crash when running :pipe -m less](https://todo.sr.ht/~rjarry/aerc/287)
- [#288: "could not MessageInfo ... NextPart: EOF" on a specific email](https://todo.sr.ht/~rjarry/aerc/288)
- [#294: Sender is not decoded in message view](https://todo.sr.ht/~rjarry/aerc/294)

## [0.18.2](https://git.sr.ht/~rjarry/aerc/refs/0.18.2) - 2024-07-29

### Fixed

- Builtin `calendar` filter error with non-GNU Awk.
- Detection of unicode width measurements on tmux 3.4.
- Dropping of events during large pastes.
- Home and End key decoding for the st terminal.

## [0.18.1](https://git.sr.ht/~rjarry/aerc/refs/0.18.1) - 2024-07-15

### Fixed

- Startup error if `log-file` directory does not exist.
- Aerc is now less pedantic about invalid headers for the maildir and notmuch
  backends.
- Error when trying to configure `smtp-domain` with STARTTLS enabled.
- `smtp-domain` is now properly taken into account for TLS connections.

## [0.18.0](https://git.sr.ht/~rjarry/aerc/refs/0.18.0) - 2024-07-02

### Added

- Add `[ui].msglist-scroll-offset` option to set a scroll offset for the
  message list.
- Add new `:align` command to align the selected message at the top, center, or
  bottom of the message list.
- Inline image previews when no filter is defined for `image/*` and the
  terminal supports it.
- `:bounce` command to reintroduce messages into the transport system.
- Message counts are available in statusline templates.
- Execute IPC commands verbatim by providing the command and its args as a
  single argument in the shell.
- Virtually any key binding can now be configured in `binds.conf`, including
  Shift+Alt+Control modifier combinations.
- Configure default message list `:split` or `:vsplit` on startup with
  `message-list-split` in `aerc.conf`.
- Create notmuch named queries with the `:query` command.
- Specify a ":q" alias for quit.
- The `:detach` command now understands globs similar to `:attach`.
- Match filters on filename via `.filename,~<regexp> =`.
- Tell aerc how to handle file-based operations on multi-file notmuch messages
  with the account config option `multi-file-strategy` and the `-m` flag to
  `:archive`, `:copy`, `:delete`, and `:move`.
- Add `[ui].dialog-{position,width,height}` to set the position, width and
  height of popover dialogs.
- New `pgp-self-encrypt` option in `accounts.conf`.
- Add `--no-ipc` flag to run `aerc mailto:...`, `aerc mbox:...`, and `aerc
  :<command...>` within the current aerc instance and prevent listening for IPC
  calls from other aerc instances.
- Add config options `disable-ipc-mailto` and `disable-ipc-mbox` to make
  `mailto:...` and `mbox:...` commands always run in a new aerc instance.
- Set global options in `accounts.conf` by placing them at the top of the file.
- Silently close the terminal tab after piping a message to a command with
  `:pipe -s <cmd>`.
- New `tag-modified` hook for notmuch and JMAP accounts.
- New `flag-changed` hook.
- Notmuch search term completions to `:query`.
- Notmuch completions for `:cf`, `:filter` and `:search`.
- Add `imaps+insecure` to the available protocols, for connections that should
  ignore issues with certificate verification.
- Add `[ui].select-last-message` option to position cursor at the bottom of the
  view.
- Propagate terminal bell from the built-in terminal.
- Added `AERC_FOLDER_ROLE` to hooks that have `AERC_FOLDER`.
- Added `{{.AccountBackend}}` to templates.
- Added `AERC_ACCOUNT_BACKEND` to hooks with `AERC_ACCOUNT`.
- Per folder key bindings can now be defined for the message viewer.
- Allow using existing directory name with `:query -f`.
- Allow specifying the folder to delete with `:rmdir`.
- The address book is now used for `:cc`, `:bcc` and `:forward`.
- Allow fallback to threading by subject with `[ui].threading-by-subject`.

### Fixed

- Calendar responses now ignore case.
- Allow account- and folder-specific binds to coexist.
- Fixed crash when running `:send` with a `:preview` tab focused.
- Deadlock when running `aerc mailto:foo@bar.com` without another instance of
  aerc already running.
- Prevent a freeze for large-scale deletions with IMAP.
- `Mime-Version` is no longer inserted in signed text parts headers. MTAs
  normalizing header case will not corrupt signatures anymore.
- Restore previous behaviour of the new message bell which was broken in the
  last two releases for at least some setups.

### Changed

- The default `[ui]` settings and the `default` styleset have changed
  extensively. A no-color theme can be restored with the `monochrome` styleset.
- The default `colorize` theme has been changed to use the base terminal colors.
- The `[viewer]` section of stylesets now preserve default values as documented
  in `aerc-stylesets(7)` unless explicitly overridden.
- Add Message-ID to the variables of `[hooks].mail-received`.
- The `TrayInfo` template variable now includes a visual mark mode indicator.
- The `disable-ipc` option in `aerc.conf` completely disables IPC.
- Improved readability of the builtin `calendar` filter.
- `:open` commands now preserve the original filename.
- Unparsable accounts are skipped, instead of aerc exiting with an error.

### Deprecated

- Built-in descriptions for the default keybinds shown on the review screen
  will be deprecated in a future release. Descriptions can be added to those
  keybinds with inline comments in binds.conf.

## [0.17.0](https://git.sr.ht/~rjarry/aerc/refs/0.17.0) - 2024-02-01

### Added

- New `flagged` criteria for `:sort`.
- New `:send-keys` command to control embedded terminals.
- Account aliases now support fnmatch-style wild cards.
- New `:suspend` command bound to `<C-z>` by default.
- Disable parent context bindings by declaring them empty.
- Toggle folding with `:fold -t`.
- `mail-deleted` hook that triggers when a message is removed/moved from a
  folder.
- `mail-added` hook that triggers when a message is added to a folder.
- Improved command completion.
- Customize key to trigger completion with `$complete` in `binds.conf`.
- Setting `complete-min-chars=manual` in `aerc.conf` now disables automatic
  completion, leaving only manually triggered completion.
- `.ThreadUnread` is now available in templates.
- Allow binding commands to `Alt+<number>` keys.
- `AERC_ACCOUNT` and `AERC_ADDRESS_BOOK_CMD` are now defined in the editor's
  environment when composing a message.
- Reply with a different account than the current one with `:reply -A
  <account>`.
- New `[ui].tab-title-viewer` setting to configure the message viewer tab title.
- The `{{.Subject}}` template is evaluated to the new option
  `[ui].empty-subject` if the subject is empty.
- Change to a folder of another account with `:cf -a <account> <folder>`.
- Patch management with `:patch`.
- Add file path to messages in templates as `{{.Filename}}`.
- New `:menu` command to invoke other ex-commands based on a shell command
  output.
- CLI flags to override paths to config files.
- Automatically attach signing key with `pgp-attach-key` in `accounts.conf`.
- Copy messages across accounts with `:cp -a <account> <folder>`.
- Move messages across accounts with `:mv -a <account> <folder>`.
- Support the `draft` flag.
- Thread arrow prefixes are now fully configurable.

### Fixed

- `colorize` support for wild cards `?` and `*`.
- Selection of headers in composer after `:compose -e` followed by `:edit -E`.
- Don't lose child messages of non-queried parents in notmuch threads
- Notmuch folders defined by the query `*` handle search, filter, and unread
  counts correctly.

### Changed

- `:open` commands are now executed with `sh -c`.
- `:pipe` commands are now executed with `sh -c`.
- Message viewer tab titles will now show `(no subject)` if there is no subject
  in the viewed email.
- Signature placement is now controlled via the `{{.Signature}}` template
  variable and not hard coded.

## [0.16.0](https://git.sr.ht/~rjarry/aerc/refs/0.16.0) - 2023-09-27

### Added

- JMAP support.
- The new account wizard now supports all source and outgoing backends.
- Edit email headers directly in the text editor with `[compose].edit-headers`
  in `aerc.conf` or with the `-e` flag for all compose related commands (e.g.
  `:compose`, `:forward`, `:recall`, etc.).
- Use `:save -A` to save all the named parts, not just attachments.
- The `<Backspace>` key can now be bound.
- `colorize` can style diff chunk function names with `diff_chunk_func`.
- Warn before sending emails with an empty subject with `empty-subject-warning`
  in `aerc.conf`.
- IMAP now uses the delimiter advertised by the server.
- `carddav-query` utility to use for `address-book-cmd`.
- Folder name mapping with `folder-map` in `accounts.conf`.
- Use `:open -d` to automatically delete temporary files.
- Remove headers from the compose window with `:header -d <name>`.
- `:attach -r <name> <cmd>` to pipe the attachments from a command.
- New `msglist_gutter` and `msglist_pill` styles for message list scrollbar.
- New `%f` placeholder to `file-picker-cmd` which expands to a location of a
  temporary file from which selected files will be read instead of the standard
  output.
- Save drafts in custom folders with `:postpone -t <folder>`.
- View "thread-context" in notmuch backends with `:toggle-thread-context`.

### Fixed

- `:archive` now works on servers using a different delimiter
- `:save -a` now works with multiple attachments with the same filename
- `:open` uses the attachment extension for temporary files, if possible
- memory leak when using notmuch with threading
- `:pipe <cmd>` now executes `sh -c "<cmd>"` as indicated in the man page.

### Changed

- Names formatted like "Last Name, First Name" are better supported in templates
- Composing an email is now aborted if the text editor exits with an error
  (e.g. with `vim`, abort an email with `:cq`).
- Aerc builtin filters path (usually `/usr/libexec/aerc/filters`) is now
  **prepended** to the default system `PATH` to avoid conflicts with installed
  distro binaries which have the same name as aerc builtin filters (e.g.
  `/usr/bin/colorize`).
- `:export-mbox` only exports marked messages, if any. Otherwise it exports
  everything, as usual.
- The local hostname is no longer exposed in outgoing `Message-Id` headers by
  default. Legacy behaviour can be restored by setting `send-with-hostname
  = true` in `accounts.conf`.
- The notmuch bindings were replaced with internal bindings
- Aerc now has a default style for most UI elements. The `default` styleset is
  now empty. Existing stylesets will only override the default attributes if
  they are set explicitly. To reset the default style and preserve existing
  stylesets appearance, these two lines must be inserted **at the beginning**:

  ```
  *.default=true
  *.normal=true
  ```
- Openers commands are not executed in with `sh -c`.

### Deprecated

- Aerc can no longer be compiled and installed with BSD make. GNU make must be
  used instead.

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
