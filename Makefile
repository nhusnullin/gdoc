# gdoc v2 build. The module lives in go/, the binaries land in bin/.
# CGO is off in dist: a static binary is principle 1.
# GOCMD, not GO: `make release GO=vX.Y.0` is Nail's go for a release, and a
# variable named on the command line wins over the file's, so a release would
# otherwise rename the compiler.
GOCMD := cd go && go

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
	$(GOCMD) test -race ./...

# vet is the plan's other two validation commands, in one target.
vet:
	$(GOCMD) vet ./...
	cd go && test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }

# The build lines carry the client secret through LDFLAGS, so make must not echo
# them: one did, into a terminal and from there into a chat, on 2026-10-03. The
# line printed instead says whether the binary has a secret, because a build
# without one is otherwise noticed only at the first sign-in.
# TestNoRecipeLineThatCarriesTheClientSecretIsEchoed and
# TestABuildSaysWhetherItCarriesASecret in go/boundary.
SECRET_STATE = client secret $(if $(GDOC_OAUTH_CLIENT_SECRET),set,NOT set: this build cannot sign anyone in)

build:
	mkdir -p bin
	@echo "go build bin/gdoc, version $(VERSION), $(SECRET_STATE)"
	@$(GOCMD) build -trimpath -ldflags "$(LDFLAGS)" -o ../bin/gdoc ./cmd/gdoc

dist:
	mkdir -p bin
	@echo "go build the three platform binaries, version $(VERSION), $(SECRET_STATE)"
	@cd go && CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o ../bin/gdoc-darwin-arm64 ./cmd/gdoc
	@cd go && CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o ../bin/gdoc-darwin-amd64 ./cmd/gdoc
	@cd go && CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o ../bin/gdoc-windows-amd64.exe ./cmd/gdoc

# The presence app, in presence/. It is a separate program in its own Swift
# package: `make test` never builds it, and a change to it touches no Go file.
# Both targets are macOS only, which is the only platform the app is for, and
# neither runs from CI's ubuntu job.
#
# app-lint is the same line the release job runs, so the formatter's version is
# the one lint was written against and a reformat is never a release's surprise.
app:
	mkdir -p bin
	presence/bundle.sh "$(VERSION)" "$(PWD)/bin/GdocPresence.app"

app-test:
	cd presence && swift test

app-lint:
	cd presence && swift format lint --strict -r Sources Tests

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
	for f in skills/*/SKILL.md; do sed -e 's|^  release: .*|  release: $(VERSION)|' "$$f" > "$$f.new" && mv "$$f.new" "$$f"; done
	git add .claude-plugin/plugin.json skills/*/SKILL.md
	git commit -m "chore: version $(VERSION)"
	git tag "$(VERSION)"
	git push origin HEAD
	git push origin "refs/tags/$(VERSION)"

# The delivery platform: the two recipes a milestone used to be run by hand.
# `make milestone` owns a run from the worktree to the end of the review and
# writes ~/gdoc-<mNN>-board/status in three lines, which is all the team lead
# reads while it runs. The lead starts it detached and goes away:
#
#   nohup make milestone PLAN=docs/plans/<file>.md REVIEW=go > ~/milestone-launch.log 2>&1 &
#
# The log is where a refusal lands: a detached run that discarded it would be
# a run the lead never hears about. A refusal that knows the board folder also
# writes `phase: refused` into the status file.
#
# REVIEW is go or ui and may come from the plan's own `Review:` line. DRY=1
# prints every command and runs none, which is what the tests assert. IDLE
# overrides the worker's idle timeout, 20 minutes by default.
milestone:
	@.ralphex/board/milestone.sh --plan "$(PLAN)"$(if $(REVIEW), --review $(REVIEW))$(if $(IDLE), --idle $(IDLE))$(if $(CODEX), --codex)$(if $(DRY), --dry)

# `make release BRANCH=<branch>` takes a finished branch to a green pull request
# and stops there. The merge, the tag and the read-back of the published zip
# need GO=vX.Y.0, which the lead passes only after Nail's message:
#
#   make release BRANCH=gdoc-v2-m29-the-archive
#   make release BRANCH=gdoc-v2-m29-the-archive GO=v2.20.0
#
# DRY=1 prints every command and runs none, which is what the tests assert.
release:
	@.ralphex/board/release.sh --branch "$(BRANCH)"$(if $(GO), --go $(GO))$(if $(DRY), --dry)

# The platform's own suite. No model runs in it: every case is a dry run or a
# stub, so it costs a second and no tokens.
platform-test:
	.ralphex/board/tests/test_milestone.sh
	.ralphex/board/tests/test_release.sh
	.ralphex/board/tests/test_meter.sh

# The team's own check. The seats under .claude/agents/ are files, so what can
# be wrong with one is a typo in its frontmatter, a model or a tool list that
# widened past what the seat is for, or a path in the brief that moved. All
# three are cheap to check and expensive to find at the moment the seat is
# called. No model runs here either.
team-check:
	.ralphex/board/tests/test_team.sh

.PHONY: test vet build dist app app-test app-lint tag milestone release platform-test team-check
