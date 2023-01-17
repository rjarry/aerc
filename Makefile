.POSIX:
.SUFFIXES:
.SUFFIXES: .1 .5 .7 .1.scd .5.scd .7.scd

VERSION!=git describe --long --abbrev=12 --tags --dirty 2>/dev/null || echo 0.14.0
VPATH=doc
PREFIX?=/usr/local
BINDIR?=$(PREFIX)/bin
SHAREDIR?=$(PREFIX)/share/aerc
LIBEXECDIR?=$(PREFIX)/libexec/aerc
MANDIR?=$(PREFIX)/share/man
GO?=go
GOFLAGS?=
BUILD_OPTS?=-trimpath
flags!=echo -- $(GOFLAGS) | base64 | tr -d '\n'
# ignore environment variable
GO_LDFLAGS:=
GO_LDFLAGS+=-X main.Version=$(VERSION)
GO_LDFLAGS+=-X main.Flags=$(flags)
GO_LDFLAGS+=-X git.sr.ht/~rjarry/aerc/config.shareDir=$(SHAREDIR)
GO_LDFLAGS+=-X git.sr.ht/~rjarry/aerc/config.libexecDir=$(LIBEXECDIR)
GO_LDFLAGS+=$(GO_EXTRA_LDFLAGS)

GOSRC!=find * -name '*.go' | grep -v filters/wrap.go
GOSRC+=go.mod go.sum

DOCS := \
	aerc.1 \
	aerc-search.1 \
	aerc-accounts.5 \
	aerc-binds.5 \
	aerc-config.5 \
	aerc-imap.5 \
	aerc-maildir.5 \
	aerc-sendmail.5 \
	aerc-notmuch.5 \
	aerc-smtp.5 \
	aerc-tutorial.7 \
	aerc-templates.7 \
	aerc-stylesets.7

all: aerc wrap $(DOCS)

build_cmd:=$(GO) build $(BUILD_OPTS) $(GOFLAGS) -ldflags "$(GO_LDFLAGS)" -o aerc

# the following command outputs nothing, we only want to execute it once
# and force .aerc.d to be regenerated when build_cmd has changed
_!=grep -sqFx '$(build_cmd)' .aerc.d || rm -f .aerc.d

.aerc.d:
	@echo 'GOFLAGS have changed, recompiling'
	@echo '$(build_cmd)' > $@

aerc: $(GOSRC) .aerc.d
	$(build_cmd)

CC?=cc
CFLAGS?=-O2 -g

wrap: filters/wrap.c
	$(CC) $(CFLAGS) $(LDFLAGS) -o wrap filters/wrap.c

.PHONY: dev
dev:
	$(MAKE) aerc BUILD_OPTS="-trimpath -race"
	GORACE="log_path=race.log strip_path_prefix=git.sr.ht/~rjarry/aerc/" ./aerc

.PHONY: fmt
fmt:
	$(GO) run mvdan.cc/gofumpt -w .

linters.so: contrib/linters.go
	$(GO) build -buildmode=plugin -o linters.so contrib/linters.go

.PHONY: lint
lint: linters.so
	@contrib/check-whitespace `git ls-files ':!:filters/vectors'` && \
		echo white space ok.
	@$(GO) run mvdan.cc/gofumpt -d . | grep ^ \
		&& echo The above files need to be formatted, please run make fmt && exit 1 \
		|| echo all files formatted.
	$(GO) run github.com/golangci/golangci-lint/cmd/golangci-lint run

.PHONY: vulncheck
vulncheck:
	$(GO) run golang.org/x/vuln/cmd/govulncheck@latest ./...

.PHONY: tests
tests: wrap
	$(GO) test $(GOFLAGS) ./...
	filters/test.sh

.PHONY: debug
debug: aerc.debug
	@echo 'Run `./aerc.debug` and use this command in another terminal to attach a debugger:'
	@echo '    dlv attach $$(pidof aerc.debug)'

aerc.debug: $(GOSRC)
	$(GO) build $(GOFLAGS) -gcflags=*=-N -gcflags=*=-l -ldflags="$(GO_LDFLAGS)" -o aerc.debug

.1.scd.1:
	scdoc < $< > $@

.5.scd.5:
	scdoc < $< > $@

.7.scd.7:
	scdoc < $< > $@

doc: $(DOCS)

# Exists in GNUMake but not in NetBSD make and others.
RM?=rm -f

clean:
	$(RM) $(DOCS) aerc wrap

install: $(DOCS) aerc wrap
	mkdir -m755 -p $(DESTDIR)$(BINDIR) $(DESTDIR)$(MANDIR)/man1 $(DESTDIR)$(MANDIR)/man5 $(DESTDIR)$(MANDIR)/man7 \
		$(DESTDIR)$(SHAREDIR) $(DESTDIR)$(SHAREDIR)/filters $(DESTDIR)$(SHAREDIR)/templates $(DESTDIR)$(SHAREDIR)/stylesets \
		$(DESTDIR)$(PREFIX)/share/applications $(DESTDIR)$(LIBEXECDIR)/filters
	install -m755 aerc $(DESTDIR)$(BINDIR)/aerc
	install -m644 aerc.1 $(DESTDIR)$(MANDIR)/man1/aerc.1
	install -m644 aerc-search.1 $(DESTDIR)$(MANDIR)/man1/aerc-search.1
	install -m644 aerc-accounts.5 $(DESTDIR)$(MANDIR)/man5/aerc-accounts.5
	install -m644 aerc-binds.5 $(DESTDIR)$(MANDIR)/man5/aerc-binds.5
	install -m644 aerc-config.5 $(DESTDIR)$(MANDIR)/man5/aerc-config.5
	install -m644 aerc-imap.5 $(DESTDIR)$(MANDIR)/man5/aerc-imap.5
	install -m644 aerc-maildir.5 $(DESTDIR)$(MANDIR)/man5/aerc-maildir.5
	install -m644 aerc-sendmail.5 $(DESTDIR)$(MANDIR)/man5/aerc-sendmail.5
	install -m644 aerc-notmuch.5 $(DESTDIR)$(MANDIR)/man5/aerc-notmuch.5
	install -m644 aerc-smtp.5 $(DESTDIR)$(MANDIR)/man5/aerc-smtp.5
	install -m644 aerc-tutorial.7 $(DESTDIR)$(MANDIR)/man7/aerc-tutorial.7
	install -m644 aerc-templates.7 $(DESTDIR)$(MANDIR)/man7/aerc-templates.7
	install -m644 aerc-stylesets.7 $(DESTDIR)$(MANDIR)/man7/aerc-stylesets.7
	install -m644 config/accounts.conf $(DESTDIR)$(SHAREDIR)/accounts.conf
	install -m644 config/aerc.conf $(DESTDIR)$(SHAREDIR)/aerc.conf
	install -m644 config/binds.conf $(DESTDIR)$(SHAREDIR)/binds.conf
	install -m755 filters/calendar $(DESTDIR)$(LIBEXECDIR)/filters/calendar
	install -m755 filters/colorize $(DESTDIR)$(LIBEXECDIR)/filters/colorize
	install -m755 filters/hldiff $(DESTDIR)$(LIBEXECDIR)/filters/hldiff
	install -m755 filters/html $(DESTDIR)$(LIBEXECDIR)/filters/html
	install -m755 filters/html-unsafe $(DESTDIR)$(LIBEXECDIR)/filters/html-unsafe
	install -m755 filters/plaintext $(DESTDIR)$(LIBEXECDIR)/filters/plaintext
	install -m755 filters/show-ics-details.py $(DESTDIR)$(LIBEXECDIR)/filters/show-ics-details.py
	install -m755 wrap $(DESTDIR)$(LIBEXECDIR)/filters/wrap
	install -m644 templates/new_message $(DESTDIR)$(SHAREDIR)/templates/new_message
	install -m644 templates/quoted_reply $(DESTDIR)$(SHAREDIR)/templates/quoted_reply
	install -m644 templates/forward_as_body $(DESTDIR)$(SHAREDIR)/templates/forward_as_body
	install -m644 stylesets/default $(DESTDIR)$(SHAREDIR)/stylesets/default
	install -m644 stylesets/dracula $(DESTDIR)$(SHAREDIR)/stylesets/dracula
	install -m644 stylesets/nord $(DESTDIR)$(SHAREDIR)/stylesets/nord
	install -m644 stylesets/pink $(DESTDIR)$(SHAREDIR)/stylesets/pink
	install -m644 stylesets/blue $(DESTDIR)$(SHAREDIR)/stylesets/blue
	install -m644 contrib/aerc.desktop $(DESTDIR)$(PREFIX)/share/applications/aerc.desktop

.PHONY: checkinstall
checkinstall:
	$(DESTDIR)$(BINDIR)/aerc -v
	test -e $(DESTDIR)$(MANDIR)/man1/aerc.1
	test -e $(DESTDIR)$(MANDIR)/man5/aerc-accounts.5
	test -e $(DESTDIR)$(MANDIR)/man5/aerc-binds.5
	test -e $(DESTDIR)$(MANDIR)/man5/aerc-config.5
	test -e $(DESTDIR)$(MANDIR)/man5/aerc-imap.5
	test -e $(DESTDIR)$(MANDIR)/man5/aerc-notmuch.5
	test -e $(DESTDIR)$(MANDIR)/man5/aerc-sendmail.5
	test -e $(DESTDIR)$(MANDIR)/man5/aerc-smtp.5
	test -e $(DESTDIR)$(MANDIR)/man7/aerc-tutorial.7
	test -e $(DESTDIR)$(MANDIR)/man7/aerc-templates.7

RMDIR_IF_EMPTY:=sh -c '! [ -d $$0 ] || ls -1qA $$0 | grep -q . || rmdir $$0'

uninstall:
	$(RM) $(DESTDIR)$(BINDIR)/aerc
	$(RM) $(DESTDIR)$(MANDIR)/man1/aerc.1
	$(RM) $(DESTDIR)$(MANDIR)/man1/aerc-search.1
	$(RM) $(DESTDIR)$(MANDIR)/man5/aerc-accounts.5
	$(RM) $(DESTDIR)$(MANDIR)/man5/aerc-binds.5
	$(RM) $(DESTDIR)$(MANDIR)/man5/aerc-config.5
	$(RM) $(DESTDIR)$(MANDIR)/man5/aerc-imap.5
	$(RM) $(DESTDIR)$(MANDIR)/man5/aerc-maildir.5
	$(RM) $(DESTDIR)$(MANDIR)/man5/aerc-sendmail.5
	$(RM) $(DESTDIR)$(MANDIR)/man5/aerc-notmuch.5
	$(RM) $(DESTDIR)$(MANDIR)/man5/aerc-smtp.5
	$(RM) $(DESTDIR)$(MANDIR)/man7/aerc-tutorial.7
	$(RM) $(DESTDIR)$(MANDIR)/man7/aerc-templates.7
	$(RM) $(DESTDIR)$(MANDIR)/man7/aerc-stylesets.7
	$(RM) -r $(DESTDIR)$(SHAREDIR)
	$(RM) -r $(DESTDIR)$(LIBEXECDIR)
	${RMDIR_IF_EMPTY} $(DESTDIR)$(BINDIR)
	$(RMDIR_IF_EMPTY) $(DESTDIR)$(MANDIR)/man1
	$(RMDIR_IF_EMPTY) $(DESTDIR)$(MANDIR)/man5
	$(RMDIR_IF_EMPTY) $(DESTDIR)$(MANDIR)/man7
	$(RMDIR_IF_EMPTY) $(DESTDIR)$(MANDIR)
	$(RM) $(DESTDIR)$(PREFIX)/share/applications/aerc.desktop
	$(RMDIR_IF_EMPTY) $(DESTDIR)$(PREFIX)/share/applications

.PHONY: gitconfig
gitconfig:
	git config format.subjectPrefix "PATCH aerc"
	git config sendemail.to "~rjarry/aerc-devel@lists.sr.ht"
	git config sendemail.validate true
	@mkdir -p .git/hooks
	ln -sf ../../contrib/sendemail-validate .git/hooks/sendemail-validate

.PHONY: check-patches
check-patches:
	@contrib/check-patches origin/master..

.PHONY: all doc clean install uninstall debug
