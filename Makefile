aerc:
	go build -o aerc

%.1: doc/%.1.scd
	scdoc < $< > $@

%.5: doc/%.5.scd
	scdoc < $< > $@

DOCS := \
	aerc.1 \
	aerc-config.5 \
	aerc-imap.5 \
	aerc-smtp.5

all: aerc $(DOCS)

clean:
	rm -f *.1 *.5 aerc

install:
	# TODO: install binary, man pages, example config, and filters from contrib

.DEFAULT_GOAL := all

.PHONY: aerc clean install
