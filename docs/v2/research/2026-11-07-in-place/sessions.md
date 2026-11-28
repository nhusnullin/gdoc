# How Nail works on published notes: session evidence

Scope: Claude Code transcripts in `~/.claude/projects/`. Window: 2026-09-26 to 2026-11-07, widened to 2026-08-29 where the window was thin. Session ids are the first 8 characters of the .jsonl file name.

## Corpus size (counted)

- Hub folders (Platform-hub, eagle, retail-card, intelligence-hub, personal-hub, ai-polish): 29 sessions with typed prompts since 2026-07-28. 15 of them since 09-26.
- Sessions that ran any `gdoc` command in those folders: 6 in total (09-10 b34ef97f, 09-17 544191fa, 09-18 79cac5d7, 09-28 45f7b48e, 10-20 543fde4f, 11-05 1d0afdfc). 3 of 15 since 09-26.
- The gdoc repo folder is mostly automated runs. Nail typed only about 56 real prompts there since 09-26 and 369 before. The recent window is thin, so I used 08-29 to 09-26 as well.
- Nail dictates by voice. Many prompts show speech-to-text slips. He rarely names a Doc by title.

## 1. Note and Doc changed in the same session

Counted: 3 sessions out of 6 gdoc sessions changed both the md and the Doc. In the window, 2 of 3.

- 543fde4f (retail-card folder, 10-20 to 10-22). The main example. Claude proposed suggestions in the Doc and edited chapter md in the same turn. Quote Claude: "I added the new sentence ... as a suggestion ... I also changed Ch 2 in the hub."
- 1d0afdfc (Platform-hub, 11-05 to 11-06). Nail wrote an answers note (35 md edits), said "im finished review - publish document to [folder]". Claude published at 18:41. At 18:59 Nail said "look into yanas comment on question 12, incirporate and propose chnages in document". Claude changed the note and made 2 suggestions.
- b34ef97f (09-10 to 09-11, before the window). Published at 23:53, started `gdoc-review live` at 00:09. 6 pieces of work handled, 8 propose calls. Claude: "New suggestion on the sentence, and the same in the paper's markdown."

How he keeps them in step:
- Publish once, then review live (2 of 3).
- Suggest in the Doc, he accepts by hand in Google, then he tells Claude to mirror it. 10-22 06:33: "i accepted all proposed changes, now your turn, apply changes in hub folder".
- He wants the Doc to win: "consider google doc 0.10 as source of truth ... you avoid that on next compile cycle i will see outdated info".
- `gdoc-align` and `gdoc-export` were never run in any hub session. In 543fde4f Claude considered align and refused: the paired file is a compiled output "never edited by hand".

Structure matters. For the retail card, the Doc is a compile of many chapter files (`compile.py`), published as versions (v0.10, v0.12, v0.20). A change in the Doc must go to 4 chapter files, glossary and rules, not to one paired note. 10-22: Claude diffed the Doc against what the hub would compile, paragraph by paragraph, and changed 4 chapters.

## 2. Kind and size of changes to a published Doc

From 543fde4f (about 33 typed prompts, my count of Claude's reports):
- Rephrase of one sentence, most common: "cannot be an opening", Apple Pay sentence, refund sentence, currency sentence. 1 to 3 sentences each.
- 14 suggestions in one batch (new opening sentence under every heading), 10-20 13:09.
- Delete a sentence ("four conditions that Compliance set"), then a paragraph.
- Glossary entry and fee explanation (4 sentences).
- Scope change across the paper: "Commercial Partner as Crypto Provider is out". Claude found 17 places in 9 files. 5 Doc suggestions, then 4 chapters.
- Rename the product: "Embedded self-custodial card". 25 replacements for "card scheme licence" the same way.
- Word fix in the Doc by a colleague (Yana's 11 comments), Nail decides, Claude applies.

In 1d0afdfc: Claude rewrote the answer to question 12 from Yana's comment. 2 suggestions. Nail's own edits there were prose changes (table to prose, Q3).

No table or section move was asked on a published Doc in the window. Sizes: 1 to 14 suggestions per request. Nail also asks the same change in many places (mass replace).

## 3. Lost time and annoyance

- Suggestion limits cost turns. Claude (10-20 13:05): "a suggestion cannot add a new paragraph". 10-22: "can't delete a whole paragraph or list item". Workarounds: replace the first sentence, or leave a comment.
- Collision with colleague's pending suggestion. 10-20 13:09: 2.2 "refused, and nothing was sent: Yana's pending change ... splits the words". Fell back to a comment.
- Accept/withdraw race. 10-22 06:09: "The withdraw failed. You had accepted the 5.1 suggestion in the meantime". 10-20 15:07: Nail changed his mind, but he had already accepted the first one, so Claude stacked a second suggestion on top. 5 withdraw calls in 543fde4f.
- Accepting by hand: Nail must open the Doc, accept, return and say so. Count: 7 of 42 messages in 543fde4f are "yes / ok / go" turns, plus the explicit "i accepted all".
- Confusion about Claude's wording, not about gdoc: 8 of 33 prompts in 543fde4f ("what does it mean", "hmm what is it about"). 10-22: "seems i already told you i dont like such phrase". 11-06: "table in Q3 is not convenient - give me prose". 09-28: "its difficult to understand ... w/o opening document".

## 4. Unintended touches and "which document?"

- No case found where Claude changed a Doc or note Nail did not mean (0 in 6 gdoc sessions).
- Stale link: 10-20 12:44 Nail pasted the v0.10 link. Claude noticed "The latest share is v0.12" and told him before acting. Nail kept v0.10.
- Restraint is common. 09-28 45f7b48e: "Nothing was posted to the Google Doc." 10-20: "None of them carries an `ai` marker, so I posted nothing". Both were read-only on request.
- Claude asked before acting: 5 AskUserQuestion prompts in 543fde4f/b34ef97f (e.g. 09-10 "has never been published. Publish it as a new Google Doc?"; 10-22 "What should happen to the bold message line?").
- Fear is on record (08-29, gdoc session 529cbd85): "What I'm worried about that you ... will change my document". He wanted tests on copies.
- 11-07 (this session): "I worked under ... embedded retail card ... Now I'm already working new session. I didn't expect that you change my Google Doc."

## 5. How he names the document

Of 8 gdoc requests in hubs since 09-10:
- Full Doc URL typed or pasted: 6 (09-11 live review, 09-28, 10-20 three times, 11-06 folder URL for publish).
- Note path plus Drive folder: 1 (09-10 `/gdoc-apply note.md to folder`).
- Implicit, from earlier in the same session: 1 (11-06 18:59 "yanas comment on question 12", no URL).
- The session folder was never the way he names a Doc. In the retail-card folder he still pasted the URL.
- He always gives a Drive folder URL to publish (3 of 3 publishes).
- Claude finds the note from the Doc: 09-28 "I'll check which hub note this doc belongs to". Claude found it with no question.

## 6. After a publish

- Same session, minutes later (2 of 3 publishes): 16 min (09-10, b34ef97f), 18 min (11-06, 1d0afdfc).
- Same Doc in later sessions, found by Doc id in transcripts: counterparties Doc (1t9VDw...): 08-29, 09-01, then 09-28, which is 27 days after 09-01 (3 sessions). Retail card Doc (v0.10): 10-20 to 10-22 (one session resumed over 2 days), then it appears again 11-03 to 11-04 (12 days later). On 11-03 no gdoc command ran. Claude worked only on md and a spreadsheet, and the Doc id was only in context.
- So 2 of 4 tracked Docs came back in a new session, weeks later. That is the exact risk Nail named. The 11-03 session is a live example where the Doc should stay untouched, and it was.
- The retail card paper shows a pattern: each compile makes a new Doc version, so "the Doc" changes URL over time (v0.10, v0.12, v0.20).

## 7. Repeated asks gdoc does not do yet

- Checklists/tasks from md (75c1a39f, 11-05 to 11-07): "[] task 1 ... I expect in google doc task list", with assignee and date. Being built.
- Google Sheets: 11-05 he pasted a Sheets link; 11-03 he asked for a spreadsheet of people and question buckets; 11-06 an RFI Sheet. (Memory notes a Sheets research already.)
- Seeing what a comment says without the Doc open: 09-28 "give me self explainable plan of actions". 
- Show changes in the Doc first, then md (10-22 06:24: "show me first chnages in openned 0.10 doc, then we do chnages in md").
- Change a whole paragraph or list item, add a paragraph (suggestion limit).

## Top 8 findings ranked by evidence

1. Dual change already happens by hand every time he reviews a published Doc: 3 of 6 gdoc sessions, 2 of 3 since 09-26. The loop is suggest, he accepts in Google, Claude mirrors to md. (543fde4f, 1d0afdfc, b34ef97f)
2. Suggestions fight him: 5 failures in two sessions (can't add or delete a paragraph, colleague's pending suggestion blocks, accept-before-withdraw race, comment-only fallback). (543fde4f)
3. Most requests are small and many: one sentence each, with batches up to 14, plus mass replaces across 9 files. (543fde4f)
4. He pastes the Doc URL in 6 of 8 requests. A second path is "the Doc we just published" in the same session (1 of 8, plus 2 publish-then-review runs). Folder never names it.
5. Two of four tracked Docs were reopened weeks later in a new session (27 and 12 days). The 11-03 session had the id in context and rightly changed nothing.
6. For the retail card the Doc is a compile of many md files, versioned (v0.10, v0.12, v0.20). The "linked md note" is not one file. Claude refused align for that reason.
7. Claude has so far been careful: 0 unintended edits found, stale-link warning, "posted nothing", 5 ask-first prompts. His worry (08-29 and 11-07) is stated more than seen.
8. Claude's wording costs him more time than gdoc: 8 of 33 prompts in the main session are "what does it mean", before any tool problem.

## What I could not tell

- Whether the 11-03 and 11-04 retail card session ever planned a Doc change. The id appears but no gdoc call ran.
- How many suggestions he accepted or rejected in Google. Transcripts show only his message "i accepted all", not the clicks.
- Sessions in the `-Users-nailkhusnullin-src-altery-intelligence` and `b2b2c-product-design` folders (no transcripts), and work done outside Claude Code (Desktop chat, Cowork).
- The gdoc folder has few typed prompts after 09-26, since the lead sessions show little of what Nail wrote. Feedback there is thin. Counts of "repeated asks" are small, so treat them as estimates.
- Whether the Adapty Doc (published 11-06) was reopened later. It is one day old.
