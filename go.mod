module git.sr.ht/~sircmpwn/aerc

go 1.13

require (
	git.sr.ht/~sircmpwn/getopt v1.0.0
	github.com/creack/pty v1.1.17
	github.com/danwakefield/fnmatch v0.0.0-20160403171240-cbb64ac3d964
	github.com/ddevault/go-libvterm v0.0.0-20190526194226-b7d861da3810
	github.com/emersion/go-imap v1.2.0
	github.com/emersion/go-imap-sortthread v1.2.0
	github.com/emersion/go-maildir v0.2.0
	github.com/emersion/go-message v0.15.0
	github.com/emersion/go-pgpmail v0.1.0
	github.com/emersion/go-sasl v0.0.0-20211008083017-0b9dcfb154ac
	github.com/emersion/go-smtp v0.15.0
	github.com/fsnotify/fsnotify v1.5.1
	github.com/gdamore/tcell/v2 v2.4.0
	github.com/go-ini/ini v1.63.2
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510
	github.com/gopherjs/gopherjs v0.0.0-20190430165422-3e4dfb77656c // indirect
	github.com/imdario/mergo v0.3.12
	github.com/kyoh86/xdg v1.2.0
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/martinlindhe/base36 v1.1.0 // indirect
	github.com/mattn/go-isatty v0.0.14
	github.com/mattn/go-pointer v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.13
	github.com/miolini/datacounter v1.0.2
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pkg/errors v0.9.1
	github.com/riywo/loginshell v0.0.0-20200815045211-7d26008be1ab
	github.com/smartystreets/assertions v1.0.1 // indirect
	github.com/smartystreets/goconvey v0.0.0-20190710185942-9d28bd7c0945 // indirect
	github.com/stretchr/testify v1.4.0
	github.com/zenhack/go.notmuch v0.0.0-20211022191430-4d57e8ad2a8b
	golang.org/x/crypto v0.0.0-20210921155107-089bfa567519
	golang.org/x/net v0.0.0-20211029224645-99673261e6eb // indirect
	golang.org/x/oauth2 v0.0.0-20211028175245-ba495a64dcb5
	golang.org/x/sys v0.0.0-20211030160813-b3129d9d1021 // indirect
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/ini.v1 v1.44.0 // indirect
)

replace golang.org/x/crypto => github.com/ProtonMail/crypto v0.0.0-20200420072808-71bec3603bf3

replace github.com/zenhack/go.notmuch => github.com/brunnre8/go.notmuch v0.0.0-20201126061756-caa2daf7093c
