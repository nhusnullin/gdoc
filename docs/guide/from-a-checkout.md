# Working from a checkout

This page is for somebody developing gdoc itself. It holds the install script,
the make targets, shell completion and where the config lives. A colleague who
only uses gdoc installs from [README.md](../../README.md) instead.

## Installing

Two routes, and which one you are on depends on whether you have this
repository checked out.
The route without a checkout is in [README.md](../../README.md).

**From a checkout**, which is how the tool is developed:

```bash
./install.sh
```

It builds `bin/gdoc` with `make build`, links it to `~/.local/bin/gdoc`, and
links all three skills, `skills/gdoc-review`, `skills/gdoc-publish` and
`skills/gdoc-restyle`, into `~/.claude/skills/`. Linked, not copied, so
`make build` refreshes the command and an edit to a skill is live with no
reinstall. It is safe to re-run: every step checks what is there first, and it
refuses to replace a real directory whose contents differ rather than write
over work that exists nowhere else. It prints the commit it installed from, and
`+ uncommitted changes` when the tree is dirty.

It writes `bin/gdoc.zsh`, the zsh completion, again on every run, or warns
saying what stopped it. The binary writes it, so a script this run wrote is the
table the parser reads. When the write fails the earlier run's script is still
on disk, and the summary says it was not rewritten this run. `.zshrc` is never
edited: the summary says `sourced from ~/.zshrc already`, or prints the one
`source` line to add yourself, or points at the warning when nothing was
written.

The script never touches `~/.config/gdoc-agent/`. Your token and your config
are written by `gdoc auth login` and by nothing else. After installing, run
`gdoc auth status` to see whether you are signed in.

`~/.local/bin` is not on the macOS default PATH. The script says so when it is
missing and tells you the line to add.

## Building

```bash
make build   # bin/gdoc, for this machine
make dist    # bin/gdoc-darwin-arm64, -darwin-amd64, -windows-amd64.exe
```

## Help and completion

```bash
bin/gdoc help              # every command, one sentence each
bin/gdoc help propose      # the words one command takes, its flags, an example
bin/gdoc completion zsh --out ~/.gdoc-completion.zsh
```

`help` prints the command table: the words each command takes, every flag with
a sentence saying what it is for, and one example. `--help` and `-h` mean the
same thing anywhere on the line. `completion` renders that same table as a
shell script and writes it to the file you name, for `zsh` or for `bash`. An
existing file is refused without `--force`, so the path above is not the
`bin/gdoc.zsh` that `install.sh` owns. Both read the one table the parser
reads, so neither offers a word or a flag the table does not hold. The bash
script has one rough edge: it sets `complete -o filenames` for the whole
command, because scoping that to the one arm offering file names needs
`compopt` and macOS ships bash 3.2, which has none. So a command word that
matches a directory in the folder you are standing in gets a trailing slash,
and the binary then refuses it by name.

## Moving the config

`GDOC_CONFIG_DIR` moves both files somewhere else, which is what the Go test
suite uses so tests never touch your real config.
