# gdocgo: a throwaway Go renderer

Built on 29 August 2026 to answer one question: can a Go port of `gdoc`'s render
path match the Python one pixel for pixel? The answer, and how it was measured,
is in [../docs/superpowers/specs/2026-08-29-go-render-spike.md](../docs/superpowers/specs/2026-08-29-go-render-spike.md).

**This is spike code.** It renders and it publishes. It does not review, reply,
export, capture, restyle, pair or keep a baseline. Do not build on it without
reading the spec's "what is not proven" section first.

## Layout

| Path | Holds |
|---|---|
| `internal/docx` | a .docx as what it is: an ordered list of zip parts |
| `internal/ooxml` | element helpers, plus the constants measured from the template |
| `internal/frontmatter` | the YAML block at the top of a note |
| `internal/shell` | the surgery on the copied master |
| `internal/body` | goldmark's AST turned into OOXML |
| `internal/contents` | the contents list, written rather than left to a field |
| `internal/pagination` | which page each heading landed on, read from the PDF outline |
| `internal/drive` | the Drive client, and the guard around it |
| `internal/generate` | the two-pass publish |
| `cmd/gdocgo` | the CLI |
| `cmd/pixdiff` | the measuring instrument: two directories of page images in, a per-page difference count out |
| `testdata/docs` | the six documents everything was measured against |

## Running it

```bash
go test ./...

go build -o /tmp/gdocgo ./cmd/gdocgo
/tmp/gdocgo build testdata/docs/03-policy.md \
  --template ../gdoc/templates/altery-group-policy-v1.0/template.docx \
  --out /tmp/out.docx

/tmp/gdocgo generate testdata/docs/03-policy.md \
  --template ../gdoc/templates/altery-group-policy-v1.0/template.docx \
  --folder <drive folder id> --name "Some Policy" --out /tmp/out.docx
```

`--links` writes real hyperlinks. It is off by default, because the Python
renderer drops link destinations and leaving it off is what makes the two
outputs comparable.

Credentials are read from `~/.config/gdoc-agent/oauth-token.json`, the same file
`gdoc auth login` writes. Nothing here logs in.
