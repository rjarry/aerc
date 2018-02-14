PKGNAME = git.sr.ht/~sircmpwn/aerc2

GOPATH = $(shell pwd)/.go
PKGPATH = .go/src/$(PKGNAME)

all: aerc

.go:
	mkdir -p $(dir $(PKGPATH))
	ln -fs $(shell dirname $(GOPATH)) $(PKGPATH)

get: .go
	env GOPATH=$(GOPATH) go get -d ./...

test: .go
	env GOPATH=$(GOPATH) go test ./...

aerc: .go
	env GOPATH=$(GOPATH) go build -o $@ ./cmd/$@

clean:
	rm -rf .go aerc

.PHONY: aerc get test clean
