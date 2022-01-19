.POSIX:
.SUFFIXES:
.SUFFIXES: .1 .5 .7 .1.scd .5.scd .7.scd

_git_version=$(shell git describe --long --tags --dirty 2>/dev/null | sed 's/-/.r/;s/-/./')
ifeq ($(strip $(_git_version)),)
VERSION=0.7.1
else
VERSION=$(_git_version)
endif

VPATH=doc
PREFIX?=/usr/local
BINDIR?=$(PREFIX)/bin
SHAREDIR?=$(PREFIX)/share/aerc
MANDIR?=$(PREFIX)/share/man
GO?=go
GOFLAGS?=

GOSRC:=$(shell find . -name '*.go')
GOSRC+=go.mod go.sum

aerc: $(GOSRC)
	$(GO) build $(GOFLAGS) \
		-ldflags "-X main.Prefix=$(PREFIX) \
		-X main.ShareDir=$(SHAREDIR) \
		-X main.Version=$(VERSION)" \
		-o $@

.PHONY: fmt
fmt:
	gofmt -w .

.PHONY: checkfmt
checkfmt:
	@if [ `gofmt -l . | wc -l` -ne 0 ]; then \
		gofmt -d .; \
		echo "ERROR: source files need reformatting with gofmt"; \
		exit 1; \
	fi

aerc.conf: config/aerc.conf.in
	sed -e 's:@SHAREDIR@:$(SHAREDIR):g' > $@ < config/aerc.conf.in

debug: $(GOSRC)
	GOFLAGS="-tags=notmuch" \
	dlv debug --headless --listen localhost:4747 &>/dev/null

DOCS := \
	aerc.1 \
	aerc-search.1 \
	aerc-config.5 \
	aerc-imap.5 \
	aerc-maildir.5 \
	aerc-sendmail.5 \
	aerc-notmuch.5 \
	aerc-smtp.5 \
	aerc-tutorial.7 \
	aerc-templates.7 \
	aerc-stylesets.7

.1.scd.1:
	scdoc < $< > $@

.5.scd.5:
	scdoc < $< > $@

.7.scd.7:
	scdoc < $< > $@

doc: $(DOCS)

all: aerc aerc.conf doc

# Exists in GNUMake but not in NetBSD make and others.
RM?=rm -f

clean:
	$(RM) $(DOCS) aerc.conf aerc

$(DESTDIR)$(BINDIR)/aerc: aerc
	install -m755 -D $< $@

$(DESTDIR)$(MANDIR)/man1/%: %
	install -m644 -D $< $@

$(DESTDIR)$(MANDIR)/man5/%: %
	install -m644 -D $< $@

$(DESTDIR)$(MANDIR)/man7/%: %
	install -m644 -D $< $@

$(DESTDIR)$(SHAREDIR)/aerc.conf: aerc.conf
	install -m644 -D $< $@

$(DESTDIR)$(SHAREDIR)/%.conf: config/%.conf
	install -m644 -D $< $@

$(DESTDIR)$(SHAREDIR)/filters/%: filters/%
	install -m755 -D $< $@

$(DESTDIR)$(SHAREDIR)/templates/%: templates/%
	install -m644 -D $< $@

$(DESTDIR)$(SHAREDIR)/stylesets/default: config/default_styleset
	install -m644 -D $< $@

install: $(DESTDIR)$(BINDIR)/aerc
install: $(DESTDIR)$(MANDIR)/man1/aerc.1
install: $(DESTDIR)$(MANDIR)/man1/aerc-search.1
install: $(DESTDIR)$(MANDIR)/man5/aerc-config.5
install: $(DESTDIR)$(MANDIR)/man5/aerc-imap.5
install: $(DESTDIR)$(MANDIR)/man5/aerc-maildir.5
install: $(DESTDIR)$(MANDIR)/man5/aerc-sendmail.5
install: $(DESTDIR)$(MANDIR)/man5/aerc-notmuch.5
install: $(DESTDIR)$(MANDIR)/man5/aerc-smtp.5
install: $(DESTDIR)$(MANDIR)/man7/aerc-tutorial.7
install: $(DESTDIR)$(MANDIR)/man7/aerc-templates.7
install: $(DESTDIR)$(MANDIR)/man7/aerc-stylesets.7
install: $(DESTDIR)$(SHAREDIR)/accounts.conf
install: $(DESTDIR)$(SHAREDIR)/aerc.conf
install: $(DESTDIR)$(SHAREDIR)/binds.conf
install: $(DESTDIR)$(SHAREDIR)/filters/hldiff
install: $(DESTDIR)$(SHAREDIR)/filters/html
install: $(DESTDIR)$(SHAREDIR)/filters/plaintext
install: $(DESTDIR)$(SHAREDIR)/templates/new_message
install: $(DESTDIR)$(SHAREDIR)/templates/quoted_reply
install: $(DESTDIR)$(SHAREDIR)/templates/forward_as_body
install: $(DESTDIR)$(SHAREDIR)/stylesets/default

RMDIR_IF_EMPTY:=sh -c '\
if test -d $$0 && ! ls -1qA $$0 | grep -q . ; then \
	rmdir $$0; \
fi'

uninstall:
	$(RM) $(DESTDIR)$(BINDIR)/aerc
	$(RM) $(DESTDIR)$(MANDIR)/man1/aerc.1
	$(RM) $(DESTDIR)$(MANDIR)/man1/aerc-search.1
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
	${RMDIR_IF_EMPTY} $(DESTDIR)$(BINDIR)
	$(RMDIR_IF_EMPTY) $(DESTDIR)$(MANDIR)/man1
	$(RMDIR_IF_EMPTY) $(DESTDIR)$(MANDIR)/man5
	$(RMDIR_IF_EMPTY) $(DESTDIR)$(MANDIR)/man7
	$(RMDIR_IF_EMPTY) $(DESTDIR)$(MANDIR)

.DEFAULT_GOAL := all

.PHONY: all doc clean install uninstall debug
