# variables that can be changed by users
#
VERSION ?= $(shell git describe --long --abbrev=12 --tags --dirty 2>/dev/null || echo 0.18.1)
DATE ?= $(shell date +%Y-%m-%d)
PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
SHAREDIR ?= $(PREFIX)/share/aerc
LIBEXECDIR ?= $(PREFIX)/libexec/aerc
MANDIR ?= $(PREFIX)/share/man
GO ?= go
INSTALL ?= install
GOFLAGS ?= $(shell contrib/goflags.sh)
BUILD_OPTS ?= -trimpath
GO_LDFLAGS :=
GO_LDFLAGS += -X main.Version=$(VERSION)
GO_LDFLAGS += -X main.Date=$(DATE)
GO_LDFLAGS += -X git.sr.ht/~rjarry/aerc/config.shareDir=$(SHAREDIR)
GO_LDFLAGS += -X git.sr.ht/~rjarry/aerc/config.libexecDir=$(LIBEXECDIR)
GO_LDFLAGS += $(GO_EXTRA_LDFLAGS)
CC ?= cc
CFLAGS ?= -O2 -g

# internal variables used for automatic rules generation with macros
gosrc = $(shell find * -type f -name '*.go') go.mod go.sum
man1 = $(subst .scd,,$(notdir $(wildcard doc/*.1.scd)))
man5 = $(subst .scd,,$(notdir $(wildcard doc/*.5.scd)))
man7 = $(subst .scd,,$(notdir $(wildcard doc/*.7.scd)))
docs = $(man1) $(man5) $(man7)
cfilters = $(subst .c,,$(notdir $(wildcard filters/*.c)))
filters = $(filter-out filters/vectors filters/test.sh filters/%.c,$(wildcard filters/*))
gofumpt_tag = v0.5.0

# Dependencies are added dynamically to the "all" rule with macros
.PHONY: all
all: aerc
	@:

aerc: $(gosrc)
	$(GO) build $(BUILD_OPTS) $(GOFLAGS) -ldflags "$(GO_LDFLAGS)" -o aerc

.PHONY: dev
dev:
	$(RM) aerc
	$(MAKE) --no-print-directory aerc BUILD_OPTS="-trimpath -race"
	GORACE="log_path=race.log strip_path_prefix=git.sr.ht/~rjarry/aerc/" ./aerc

.PHONY: fmt
fmt:
	$(GO) run mvdan.cc/gofumpt@$(gofumpt_tag) -w .

.PHONY: lint
lint:
	@contrib/check-whitespace `git ls-files ':!:filters/vectors'` && \
		echo white space ok.
	@contrib/check-docs && echo docs ok.
	@$(GO) run mvdan.cc/gofumpt@$(gofumpt_tag) -d . | grep ^ \
		&& echo The above files need to be formatted, please run make fmt && exit 1 \
		|| echo all files formatted.
	$(GO) run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.56.1 run \
		$$(echo $(GOFLAGS) | sed s/-tags=/--build-tags=/)
	$(GO) run $(GOFLAGS) contrib/linters.go ./...

.PHONY: vulncheck
vulncheck:
	$(GO) run golang.org/x/vuln/cmd/govulncheck@latest ./...

.PHONY: tests
tests: $(cfilters)
	$(GO) test $(GOFLAGS) ./...
	filters/test.sh

.PHONY: debug
debug: aerc.debug
	@echo 'Run `./aerc.debug` and use this command in another terminal to attach a debugger:'
	@echo '    dlv attach $$(pidof aerc.debug)'

aerc.debug: $(gosrc)
	$(GO) build $(subst -trimpath,,$(GOFLAGS)) -gcflags=*=-N -gcflags=*=-l -ldflags="$(GO_LDFLAGS)" -o aerc.debug

.PHONY: doc
doc: $(docs)
	@:

.PHONY: clean
clean:
	$(RM) $(docs) aerc $(cfilters) linters.so

# Dependencies are added dynamically to the "install" rule with macros
.PHONY: install
install:
	@:

.PHONY: checkinstall
checkinstall:
	$(DESTDIR)$(BINDIR)/aerc -v
	for m in $(man1); do test -e $(DESTDIR)$(MANDIR)/man1/$$m || exit; done
	for m in $(man5); do test -e $(DESTDIR)$(MANDIR)/man5/$$m || exit; done
	for m in $(man7); do test -e $(DESTDIR)$(MANDIR)/man7/$$m || exit; done

.PHONY: uninstall
uninstall:
	@echo $(installed) | tr ' ' '\n' | sort -ru | while read -r f; do \
		echo rm -f $$f && rm -f $$f || exit; \
	done
	@echo $(dirs) | tr ' ' '\n' | sort -ru | while read -r d; do \
		if [ -d $$d ] && ! ls -Aq1 $$d | grep -q .; then \
			echo rmdir $$d && rmdir $$d || exit; \
		fi; \
	done

.PHONY: gitconfig
gitconfig:
	git config format.subjectPrefix "PATCH aerc"
	git config sendemail.to "~rjarry/aerc-devel@lists.sr.ht"
	git config format.notes true
	git config notes.rewriteRef refs/notes/commits
	git config notes.rewriteMode concatenate
	@mkdir -p .git/hooks
	@rm -f .git/hooks/commit-msg*
	ln -s ../../contrib/commit-msg .git/hooks/commit-msg
	@rm -f .git/hooks/sendemail-validate*
	@if grep -q GIT_SENDEMAIL_FILE_COUNTER `git --exec-path`/git-send-email 2>/dev/null; then \
		set -xe; \
		ln -s ../../contrib/sendemail-validate .git/hooks/sendemail-validate && \
		git config sendemail.validate true; \
	fi

.PHONY: check-patches
check-patches:
	@contrib/check-patches origin/master..

.PHONY: validate
validate: CFLAGS = -Wall -Wextra -Wconversion -Werror -Wformat-security -Wstack-protector -Wpedantic -Wmissing-prototypes
validate: all tests lint check-patches

# Generate build and install rules for one man page
#
# $1: man page name (e.g: aerc.1)
#
define install_man
$1: doc/$1.scd
	scdoc < $$< > $$@

$1_section = $$(subst .,,$$(suffix $1))
$1_install_dir = $$(DESTDIR)$$(MANDIR)/man$$($1_section)
dirs += $$($1_install_dir)
installed += $$($1_install_dir)/$1

$$($1_install_dir)/$1: $1 | $$($1_install_dir)
	$$(INSTALL) -m644 $$< $$@

all: $1
install: $$($1_install_dir)/$1
endef

# Generate build and install rules for one filter
#
# $1: filter source path or name
#
define install_filter
ifneq ($(wildcard filters/$1.c),)
$1: filters/$1.c
	$$(CC) $$(CFLAGS) $$(LDFLAGS) -o $$@ $$<

all: $1
endif

$1_install_dir = $$(DESTDIR)$$(LIBEXECDIR)/filters
dirs += $$($1_install_dir)
installed += $$($1_install_dir)/$$(notdir $1)

$$($1_install_dir)/$$(notdir $1): $1 | $$($1_install_dir)
	$$(INSTALL) -m755 $$< $$@

install: $$($1_install_dir)/$$(notdir $1)
endef

# Generate install rules for any file
#
# $1: source file
# $2: mode
# $3: target dir
#
define install_file
dirs += $3
installed += $3/$$(notdir $1)

$3/$$(notdir $1): $1 | $3
	$$(INSTALL) -m$2 $$< $$@

install: $3/$$(notdir $1)
endef

# Call macros to generate build and install rules
$(foreach m,$(docs),\
	$(eval $(call install_man,$m)))
$(foreach f,$(filters) $(cfilters),\
	$(eval $(call install_filter,$f)))
$(foreach f,$(wildcard config/*.conf),\
	$(eval $(call install_file,$f,644,$(DESTDIR)$(SHAREDIR))))
$(foreach s,$(wildcard stylesets/*),\
	$(eval $(call install_file,$s,644,$(DESTDIR)$(SHAREDIR)/stylesets)))
$(foreach t,$(wildcard templates/*),\
	$(eval $(call install_file,$t,644,$(DESTDIR)$(SHAREDIR)/templates)))
$(eval $(call install_file,contrib/aerc.desktop,644,$(DESTDIR)$(PREFIX)/share/applications))
$(eval $(call install_file,aerc,755,$(DESTDIR)$(BINDIR)))
$(eval $(call install_file,contrib/carddav-query,755,$(DESTDIR)$(BINDIR)))

$(sort $(dirs)):
	mkdir -p $@

.DELETE_ON_ERROR:
