.POSIX:
.SUFFIXES:
.SUFFIXES: .1 .5 .7 .1.scd .5.scd .7.scd

VERSION=0.5.1

VPATH=doc
PREFIX?=/usr/local
BINDIR?=$(PREFIX)/bin
SHAREDIR?=$(PREFIX)/share/aerc
MANDIR?=$(PREFIX)/share/man
GO?=go
GOFLAGS?=

GOSRC!=find . -name '*.go'
GOSRC+=go.mod go.sum

aerc: $(GOSRC)
	$(GO) build $(GOFLAGS) \
		-ldflags "-X main.Prefix=$(PREFIX) \
		-X main.ShareDir=$(SHAREDIR) \
		-X main.Version=$(VERSION)" \
		-o $@

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

install: all
	mkdir -m755 -p $(DESTDIR)$(BINDIR) $(DESTDIR)$(MANDIR)/man1 $(DESTDIR)$(MANDIR)/man5 $(DESTDIR)$(MANDIR)/man7 \
		$(DESTDIR)$(SHAREDIR) $(DESTDIR)$(SHAREDIR)/filters $(DESTDIR)$(SHAREDIR)/templates $(DESTDIR)$(SHAREDIR)/stylesets
	install -m755 aerc $(DESTDIR)$(BINDIR)/aerc
	install -m644 aerc.1 $(DESTDIR)$(MANDIR)/man1/aerc.1
	install -m644 aerc-search.1 $(DESTDIR)$(MANDIR)/man1/aerc-search.1
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
	install -m644 aerc.conf $(DESTDIR)$(SHAREDIR)/aerc.conf
	install -m644 config/binds.conf $(DESTDIR)$(SHAREDIR)/binds.conf
	install -m755 filters/hldiff $(DESTDIR)$(SHAREDIR)/filters/hldiff
	install -m755 filters/html $(DESTDIR)$(SHAREDIR)/filters/html
	install -m755 filters/plaintext $(DESTDIR)$(SHAREDIR)/filters/plaintext
	install -m644 templates/quoted_reply $(DESTDIR)$(SHAREDIR)/templates/quoted_reply
	install -m644 templates/forward_as_body $(DESTDIR)$(SHAREDIR)/templates/forward_as_body
	install -m644 config/default_styleset $(DESTDIR)$(SHAREDIR)/stylesets/default

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
