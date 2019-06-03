.POSIX:
.SUFFIXES:
.SUFFIXES: .1 .5 .7 .1.scd .5.scd .7.scd
VPATH=doc
PREFIX?=/usr/local
_INSTDIR=$(DESTDIR)$(PREFIX)
BINDIR?=$(_INSTDIR)/bin
SHAREDIR?=$(_INSTDIR)/share/aerc
MANDIR?=$(_INSTDIR)/share/man
GOFLAGS?=

GOSRC!=find . -name '*.go'
GOSRC+=go.mod go.sum

aerc: $(GOSRC)
	go build $(GOFLAGS) \
		-ldflags "-X main.Prefix=$(PREFIX)" \
		-ldflags "-X main.ShareDir=$(SHAREDIR)" \
		-o $@

aerc.conf: config/aerc.conf.in
	sed -e 's:@SHAREDIR@:$(SHAREDIR):g' > $@ < config/aerc.conf.in

DOCS := \
	aerc.1 \
	aerc-config.5 \
	aerc-imap.5 \
	aerc-smtp.5 \
	aerc-tutorial.7

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
	$(RM) $(DOCS) aerc

install: all
	mkdir -p $(BINDIR) $(MANDIR)/man1 $(MANDIR)/man5 $(MANDIR)/man7 \
		$(SHAREDIR) $(SHAREDIR)/filters
	install -m755 aerc $(BINDIR)/aerc
	install -m644 aerc.1 $(MANDIR)/man1/aerc.1
	install -m644 aerc-config.5 $(MANDIR)/man5/aerc-config.5
	install -m644 aerc-imap.5 $(MANDIR)/man5/aerc-imap.5
	install -m644 aerc-smtp.5 $(MANDIR)/man5/aerc-smtp.5
	install -m644 aerc-tutorial.7 $(MANDIR)/man7/aerc-tutorial.7
	install -m644 config/accounts.conf $(SHAREDIR)/accounts.conf
	install -m644 aerc.conf $(SHAREDIR)/aerc.conf
	install -m644 config/binds.conf $(SHAREDIR)/binds.conf
	install -m755 contrib/hldiff.py $(SHAREDIR)/filters/hldiff.py
	install -m755 contrib/html $(SHAREDIR)/filters/html
	install -m755 contrib/plaintext.py $(SHAREDIR)/filters/plaintext.py

.DEFAULT_GOAL := all

.PHONY: all doc clean install
