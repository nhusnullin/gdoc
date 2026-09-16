---
worth: before the M7d branch merges
where: go/cmd/gdoc/gdoc, commit c38b096 on gdoc-v2-m7d-help-completion-skills
added: 2026-09-16
---
# The M7d branch's history carries a 14 MB built binary

A `go build` run from inside `go/cmd/gdoc/` wrote the binary as
`go/cmd/gdoc/gdoc`, and c38b096 committed it. The working tree is clean now:
the deletion is staged, `git ls-files go/cmd/gdoc/gdoc` returns nothing, and
`.gitignore` covers the path so the same build cannot put it back.

The history is not. The blob is 14,348,450 bytes in every commit from c38b096
to the tip, and `git diff --stat main...HEAD` still lists it as
`Bin 0 -> 14348450 bytes`. A commit that removes a file does not remove its
blob from the pack, so merging this branch as it stands puts 14 MB into `main`
for good. Every clone and every CI checkout pays it from then on, and after the
merge the only way out is rewriting shared history.

Nothing in the suite catches it and nothing there can: both repository walks in
`go/boundary/docs_test.go` filter on Go files and doc roots, so a binary blob
is invisible to them.

`worth: before the M7d branch merges` because the fix is a history rewrite and
that is Nail's call, not a review round's. The branch has no upstream, so a
rewrite is safe today:

```bash
git branch backup/m7d-pre-rewrite
git stash                       # the review fixes are uncommitted
git filter-branch --index-filter \
    'git rm --cached --ignore-unmatch go/cmd/gdoc/gdoc' main..HEAD
git stash pop
```

Every commit hash on the branch changes, so a ralphex board run pointing at
them has to be repointed. Keep the staged deletion and the `.gitignore` entry
either way. Doing it after the merge means rewriting `main`, which is a
different and much worse conversation.

Raised by the external review in rounds 01, 02 and 03, 2026-09-16.
