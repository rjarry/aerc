module git.sr.ht/~rjarry/aerc

go 1.18

require (
	git.sr.ht/~rockorager/go-jmap v0.3.0
	git.sr.ht/~rockorager/tcell-term v0.8.0
	git.sr.ht/~sircmpwn/getopt v1.0.0
	github.com/ProtonMail/go-crypto v0.0.0-20230417170513-8ee5748c52b5
	github.com/arran4/golang-ical v0.0.0-20230318005454-19abf92700cc
	github.com/danwakefield/fnmatch v0.0.0-20160403171240-cbb64ac3d964
	github.com/emersion/go-imap v1.2.1
	github.com/emersion/go-imap-sortthread v1.2.0
	github.com/emersion/go-maildir v0.3.0
	github.com/emersion/go-mbox v1.0.3
	github.com/emersion/go-message v0.16.0
	github.com/emersion/go-msgauth v0.6.6
	github.com/emersion/go-pgpmail v0.2.0
	github.com/emersion/go-sasl v0.0.0-20220912192320-0145f2c60ead
	github.com/emersion/go-smtp v0.16.0
	github.com/fsnotify/fsevents v0.1.1
	github.com/fsnotify/fsnotify v1.6.0
	github.com/gatherstars-com/jwz v1.4.0
	github.com/gdamore/tcell/v2 v2.6.0
	github.com/go-ini/ini v1.67.0
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510
	github.com/lithammer/fuzzysearch v1.1.5
	github.com/mattn/go-isatty v0.0.18
	github.com/mattn/go-runewidth v0.0.14
	github.com/miolini/datacounter v1.0.3
	github.com/pkg/errors v0.9.1
	github.com/rivo/uniseg v0.4.4
	github.com/riywo/loginshell v0.0.0-20200815045211-7d26008be1ab
	github.com/stretchr/testify v1.8.2
	github.com/syndtr/goleveldb v1.0.0
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e
	github.com/zenhack/go.notmuch v0.0.0-20220918173508-0c918632c39e
	golang.org/x/oauth2 v0.7.0
	golang.org/x/sys v0.7.0
	golang.org/x/tools v0.6.0
)

require (
	github.com/cloudflare/circl v1.3.2 // indirect
	github.com/creack/pty v1.1.18 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/emersion/go-textwrapper v0.0.0-20200911093747-65d896831594 // indirect
	github.com/gdamore/encoding v1.0.0 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/onsi/gomega v1.20.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.8.1 // indirect
	golang.org/x/crypto v0.8.0 // indirect
	golang.org/x/net v0.9.0 // indirect
	golang.org/x/term v0.7.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace golang.org/x/crypto => github.com/ProtonMail/crypto v0.0.0-20200420072808-71bec3603bf3

replace github.com/zenhack/go.notmuch => github.com/brunnre8/go.notmuch v0.0.0-20201126061756-caa2daf7093c
