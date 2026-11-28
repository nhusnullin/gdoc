1. **Intent: keep judgement in the skill, but require verifiable current-session evidence.** I disagree that a pasted link alone is enough. `sessions.md` includes read-only requests containing links; naming a Doc establishes the target, not permission to edit it. Publishing likewise does not authorize every later note edit.

   A cheap binary fact is whether a supplied message reference resolves to a **user-authored message in the current Claude Code transcript**, containing the target link, or to a publish result from this session. Bind that reference to the current invocation; never save it in front matter. The binary reports provenance; the skill judges whether the request actually asks for a Doc change. This requires a trustworthy session identifier and transcript access, whose availability should be checked first. A model-supplied `--intent=yes` adds nothing. Session provenance reduces the retail-card risk without pretending to prove intent.

2. **Two-sided write: keep Doc-first and the pending key, but strengthen recovery.** The paragraph comparison is a good precondition, grounded in `sync.md` and `body`’s typographer. It is not a transaction. The source can change after comparison; Claude can write different replacement words; the Doc can change after read-back.

   Record the resolved source path, contextual before/after paragraph text, and source precondition alongside the operation. Recheck the source before Claude’s patch, then render and verify the resulting paragraph before clearing pending. On mismatch, retain pending and report divergence. Finding the replacement somewhere is insufficient.

   The pending key is worth its strict-schema cost: interruption between two independent writes is unavoidable, and Principle 2 puts recovery facts in the note. Keep one unresolved operation per document; never overwrite it with another. Recovery must handle crashes before the remote write and after the note write, not just between them. Pending records are evidence, never permission to retry a Doc write.

   I disagree with point 10’s automatic Doc-only fallback: ambiguous source mapping should stop the paired operation before either write. Also remove “hub’s git history” from point 7: `PRINCIPLES.md` explicitly says the vault is not a git repository.

3. **Colour: choose off-palette light green.** I agree with the lead. `sync.md`, `house.yaml`’s debt meaning, and `export/doc.go` establish a concrete cost to house green: an editing trace becomes an author’s unresolved claim on export. Separate those meanings and test that the colours are distinguishable in practice.

   However, exact colour is not ownership. “Clear the green” must explicitly mean all passages of this edit colour, including anything a person painted identically. Do not promise to identify Claude’s passages from hex alone.

4. **Scope: start with one non-empty replacement inside one plain body paragraph.** Four kinds are too wide for a “feel” experiment. `sessions.md` identifies sentence rephrasing as the common case; one replacement gives both essential experiences: words changing green in the open Doc and the Markdown diff in Claude Code.

   Require one uniquely resolved source paragraph, allowing a compiled document only when its chapter source resolves uniquely. Defer paragraph insertion/deletion, mass replacement, and deletion-only edits. These introduce structural mapping, multi-file partial completion, or an invisible result without improving the first demonstration.

5. **Measurements: add interruption and recovery tests; narrow the live matrix.** Add lost-response simulation, termination at each write boundary, concurrent source edits, and a second invocation finding pending. Test repeated paragraphs, smart punctuation, Unicode, Markdown escaping, and post-edit rendered equality.

   Exercise intent with the actual evidence patterns: old published note in a new session, pasted link for inspection, and publish followed by an explicit edit request. Measure whether the user understands the mode and sees the paired diff without another acceptance turn.

   Combine measurements 2 and 3. Defer destructive overlap testing for excluded operations. Investigate round-trip differences relevant to plain paragraphs now; fixing all 27 is not a prerequisite for this smaller experiment.