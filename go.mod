module git.sr.ht/~sircmpwn/aerc

go 1.13

require (
	git.sr.ht/~sircmpwn/getopt v0.0.0-20190808004552-daaf1274538b
	github.com/creack/pty v1.1.10
	github.com/danwakefield/fnmatch v0.0.0-20160403171240-cbb64ac3d964
	github.com/ddevault/go-libvterm v0.0.0-20190526194226-b7d861da3810
	github.com/emersion/go-imap v1.0.6-0.20200914131512-88f167c1e6f7
	github.com/emersion/go-imap-idle v0.0.0-20190519112320-2704abd7050e
	github.com/emersion/go-imap-sortthread v1.1.0
	github.com/emersion/go-maildir v0.2.0
	github.com/emersion/go-message v0.12.1-0.20200824204225-9094bd0b8bc0
	github.com/emersion/go-pgpmail v0.0.0-20200303213726-db035a3a4139
	github.com/emersion/go-sasl v0.0.0-20200509203442-7bfe0ed36a21
	github.com/emersion/go-smtp v0.12.1
	github.com/fsnotify/fsnotify v1.4.7
	github.com/gdamore/tcell v1.3.0
	github.com/go-ini/ini v1.52.0
	github.com/golang/protobuf v1.3.4 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510
	github.com/gopherjs/gopherjs v0.0.0-20190430165422-3e4dfb77656c // indirect
	github.com/imdario/mergo v0.3.8
	github.com/kyoh86/xdg v1.2.0
	github.com/lucasb-eyer/go-colorful v1.0.3 // indirect
	github.com/mattn/go-isatty v0.0.12
	github.com/mattn/go-pointer v0.0.0-20190911064623-a0a44394634f // indirect
	github.com/mattn/go-runewidth v0.0.8
	github.com/miolini/datacounter v1.0.2
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pkg/errors v0.9.1
	github.com/riywo/loginshell v0.0.0-20190610082906-2ed199a032f6
	github.com/smartystreets/assertions v1.0.1 // indirect
	github.com/smartystreets/goconvey v0.0.0-20190710185942-9d28bd7c0945 // indirect
	github.com/stretchr/testify v1.3.0
	github.com/zenhack/go.notmuch v0.0.0-20190821052706-5a1961965cfb
	golang.org/x/crypto v0.0.0-20200302210943-78000ba7a073
	golang.org/x/net v0.0.0-20200301022130-244492dfa37a // indirect
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sys v0.0.0-20200302150141-5c8b2ff67527 // indirect
	golang.org/x/text v0.3.3 // indirect
	google.golang.org/appengine v1.6.5 // indirect
	gopkg.in/ini.v1 v1.44.0 // indirect
	gopkg.in/yaml.v2 v2.2.8 // indirect
)

replace golang.org/x/crypto => github.com/ProtonMail/crypto v0.0.0-20200420072808-71bec3603bf3

replace github.com/gdamore/tcell => git.sr.ht/~sircmpwn/tcell v0.0.0-20190807054800-3fdb6bc01a50
