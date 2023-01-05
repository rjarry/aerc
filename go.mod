module git.sr.ht/~rjarry/aerc

go 1.16

require (
	git.sr.ht/~rockorager/tcell-term v0.4.0
	git.sr.ht/~sircmpwn/getopt v1.0.0
	github.com/ProtonMail/go-crypto v0.0.0-20211221144345-a4f6767435ab
	github.com/arran4/golang-ical v0.0.0-20220517104411-fd89fefb0182
	github.com/creack/pty v1.1.18 // indirect
	github.com/danwakefield/fnmatch v0.0.0-20160403171240-cbb64ac3d964
	github.com/emersion/go-imap v1.2.0
	github.com/emersion/go-imap-sortthread v1.2.0
	github.com/emersion/go-maildir v0.2.0
	github.com/emersion/go-mbox v1.0.2
	github.com/emersion/go-message v0.15.0
	github.com/emersion/go-msgauth v0.6.5
	github.com/emersion/go-pgpmail v0.2.0
	github.com/emersion/go-sasl v0.0.0-20211008083017-0b9dcfb154ac
	github.com/emersion/go-smtp v0.15.0
	github.com/fsnotify/fsnotify v1.5.4
	github.com/gatherstars-com/jwz v1.3.2-0.20221104050604-3da8c59aef0a
	github.com/gdamore/tcell/v2 v2.5.3
	github.com/go-ini/ini v1.63.2
	github.com/golangci/golangci-lint v1.49.0
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510
	github.com/imdario/mergo v0.3.12
	github.com/kyoh86/xdg v1.2.0
	github.com/lithammer/fuzzysearch v1.1.3
	github.com/mattn/go-isatty v0.0.16
	github.com/mattn/go-runewidth v0.0.13
	github.com/miolini/datacounter v1.0.2
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pkg/errors v0.9.1
	github.com/rivo/uniseg v0.2.0
	github.com/riywo/loginshell v0.0.0-20200815045211-7d26008be1ab
	github.com/stretchr/testify v1.8.0
	github.com/syndtr/goleveldb v1.0.0
	github.com/xo/terminfo v0.0.0-20210125001918-ca9a967f8778
	github.com/zenhack/go.notmuch v0.0.0-20211022191430-4d57e8ad2a8b
	golang.org/x/oauth2 v0.0.0-20220411215720-9780585627b5
	golang.org/x/tools v0.1.12
)

replace golang.org/x/crypto => github.com/ProtonMail/crypto v0.0.0-20200420072808-71bec3603bf3

replace github.com/zenhack/go.notmuch => github.com/brunnre8/go.notmuch v0.0.0-20201126061756-caa2daf7093c
