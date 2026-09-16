# gdoc v2 build. The module lives in go/, the binaries land in bin/.
# CGO is off in dist: a static binary is principle 1.
GO := cd go && go

# The version comes from the tag, so a colleague's report names the build it
# came from and nobody has to write a number down twice.
#
# Only a tag, and only a clean one. `git describe --exact-match` names a
# version when HEAD is exactly a release and says nothing otherwise, and
# --dirty marks the tag that has uncommitted work over it, which is not that
# release either. Anything but a clean tag keeps `dev`, which main.go's
# releaseVersion reads as "no release is behind this binary" and leaves out of
# the envelope. A commit hash here would put a number in every envelope that
# names no release a colleague could fetch, and would be the one thing a
# version-gated skill cannot compare against anything.
#
# Both assignments are lazy on purpose: git runs when a recipe expands VERSION,
# which is `build` and `dist` through LDFLAGS and `tag` directly, so `make test`
# and `make vet` run no git at all. `make tag VERSION=vX.Y.0` runs none of it
# either, because a variable named on the command line wins over this one.
DESCRIBED = $(shell git describe --tags --exact-match --dirty 2>/dev/null)
VERSION = $(if $(filter-out %-dirty,$(DESCRIBED)),$(DESCRIBED),dev)

# The OAuth client secret is not in the source: the linker sets it from the
# environment, so a release build carries it and the tree never does. A build
# without it can read with a token it already holds but cannot sign anyone in.
#
# -s -w drop the symbol table and the DWARF data, which halves the file and
# takes the build machine's paths out of it. -trimpath on the build itself
# takes out the rest, so the same tag built anywhere is the same bytes and
# nobody's home directory ships to the team.
LDFLAGS = -s -w -X main.version=$(VERSION) -X gdoc/internal/auth.BundledClientSecret=$(GDOC_OAUTH_CLIENT_SECRET)

test:
	$(GO) test -race ./...

# vet is the plan's other two validation commands, in one target.
vet:
	$(GO) vet ./...
	cd go && test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }

build:
	mkdir -p bin
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o ../bin/gdoc ./cmd/gdoc

dist:
	mkdir -p bin
	cd go && CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o ../bin/gdoc-darwin-arm64 ./cmd/gdoc
	cd go && CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o ../bin/gdoc-darwin-amd64 ./cmd/gdoc
	cd go && CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o ../bin/gdoc-windows-amd64.exe ./cmd/gdoc

# `make tag VERSION=vX.Y.0` cuts a release by hand: it writes the version into
# the plugin manifest, commits, tags and pushes, and the tag is what the
# release workflow builds from.
#
# Only a minor. The nightly owns the patch numbers, so a patch cut here would
# collide with the one CI cuts at 02:00 UTC, and two tags on the same number
# are two releases a colleague cannot tell apart.
#
# The refusals come before anything is written. A rejected version leaves the
# tree exactly as it was, so a typo costs nothing but the message.
tag:
	@echo "$(VERSION)" | grep -Eq '^v[0-9]+\.[0-9]+\.0$$' || { echo "make tag: VERSION=$(VERSION) is not vX.Y.0. Patch versions are the nightly's."; exit 1; }
	@git diff --quiet HEAD || { echo "make tag: the tree has uncommitted changes. Commit them first."; exit 1; }
	@git rev-parse -q --verify "refs/tags/$(VERSION)" >/dev/null && { echo "make tag: $(VERSION) is already a tag."; exit 1; } || true
	sed -e 's|"version": "[^"]*"|"version": "$(VERSION)"|' .claude-plugin/plugin.json > .claude-plugin/plugin.json.new
	mv .claude-plugin/plugin.json.new .claude-plugin/plugin.json
	git add .claude-plugin/plugin.json
	git commit -m "chore: version $(VERSION)"
	git tag "$(VERSION)"
	git push origin HEAD
	git push origin "refs/tags/$(VERSION)"

.PHONY: test vet build dist tag
