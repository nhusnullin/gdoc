# Spikes

Working code from the design session of 2026-08-29. Not the product. Each of these
proved something that [DECISIONS.md](../DECISIONS.md) now asserts, and they are kept
so a claim can be re-run rather than re-argued.

None of it is wired into anything. All of it was written against Nail's real
template and his real Drive test folder.

## `config/` — the house style as a file

This is the evidence behind the decision that `house.yaml` replaces the `.docx`
master.

| | |
|---|---|
| `house.yaml` | the whole house style, 1,110 lines, logo included as base64 |
| `house-small.yaml` | the 164-line subset, kept only to show what the hybrid option would have looked like |
| `gen.py` | config plus markdown to a `.docx`. **Reads no master.** |
| `compare.py` | **the drift test.** 160 items, renders both ways and reports every difference |
| `make_config.py`, `extract_tables.py` | the one-time extraction from the master |
| `style-profile.json` | the same style read back out of Google, cross-checked against the master |

`compare.py` is the important one. The config decision is only safe because this
exists: it fails when a value drifts, and without it the config is a promise rather
than a checked fact. Last run: **137 of 160 identical, worst position offset 2.5pt.**

## `probes/` — what Google's API actually does

Each script establishes one fact, and several of those facts contradict the
documentation. `gauth.py` loads the stored token; the rest depend on it.

| | proves |
|---|---|
| `gate.py` | the Developer Preview is live: `SUGGEST`, `acceptSuggestion`, `commentsViewMode` |
| `inplace.py` | an anchored comment and a suggestion, made **in place**, verified through the docx export |
| `ic.py` | the undocumented `insertComment` shape: `{range, content}`, found by probing every other spelling |
| `layout.py` | what the preview did **not** unlock: no contents list, no first-page header, no page numbers, no positioned images |
| `authors.py` | authorship cannot be set by any route. Everything gdoc writes is signed Nail |
| `many.py` | twelve pending suggestions, ids stable, resolving three leaves the other nine untouched |
| `probe3.py`, `probe5.py` | the `writeMode` enum, and that no header or preview-version trick bypasses the gate |
| `read_suggestions.py` | reads suggestions back out of a document |
| `marks.py` | the retired colour scheme, kept because it is the fallback if the preview is withdrawn |

**The reliable probe, used throughout:** export the document as `.docx` and inspect
`word/comments.xml` and the `w:commentRangeStart`/`End` markers. Drive's `anchor`
and `quotedFileContent` fields survive detachment and prove nothing.

## `go/` — the rewrite is feasible

Not a survey. Every capability was built and run.

| | proves |
|---|---|
| `xmltest/` | Go's `encoding/xml` **corrupts** OOXML; `beevik/etree` round-trips the real 194 KB template with one apostrophe of difference |
| `ziptest/` | reading a `.docx` and extracting images in 95 lines, getting **3 of 3** where pandoc structurally gets 2 |
| `trktest/` | tracked changes written by hand, validated by two independent readers |
| `difftest/` | word-level diff to redline, accept and reject both restoring exactly |
| `gmtest/` | goldmark parses everything needed, including the nested-image-in-heading case; `==mark==` added in 55 lines |
| `pdftest/` | page numbers read back out of a Google Docs PDF export |
| `e2e/` | the whole path end to end |

Static binaries built clean at about 3 MB. Two dependencies, `beevik/etree` (BSD-2)
and `yuin/goldmark` (MIT). No cgo.

## Running any of it

The Python probes need the venv at `~/.config/gdoc-agent/venv/bin/python` and the
stored OAuth token. They create documents in the Drive test folder and trash them.
The Go spikes are standalone modules; `go run .` in each.
