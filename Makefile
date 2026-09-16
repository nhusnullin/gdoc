# gdoc v2 build. The module lives in go/, the binaries land in bin/.
# CGO is off in dist: a static binary is principle 1.
GO := cd go && go

# The OAuth client secret is not in the source: the linker sets it from the
# environment, so a release build carries it and the tree never does. A build
# without it can read with a token it already holds but cannot sign anyone in.
LDFLAGS := -X gdoc/internal/auth.BundledClientSecret=$(GDOC_OAUTH_CLIENT_SECRET)

test:
	$(GO) test -race ./...

# vet is the plan's other two validation commands, in one target.
vet:
	$(GO) vet ./...
	cd go && test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }

build:
	mkdir -p bin
	$(GO) build -ldflags "$(LDFLAGS)" -o ../bin/gdoc ./cmd/gdoc

dist:
	mkdir -p bin
	cd go && CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o ../bin/gdoc-darwin-arm64 ./cmd/gdoc
	cd go && CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o ../bin/gdoc-darwin-amd64 ./cmd/gdoc
	cd go && CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o ../bin/gdoc-windows-amd64.exe ./cmd/gdoc

.PHONY: test vet build dist
