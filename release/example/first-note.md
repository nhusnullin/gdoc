---
title: Vendor Access Review
doc_type: Procedure
version: 1.0
date: 2026-09-16
owner: Head of Operations
classification: internal
revisions:
  - version: 1.0
    date: 2026-09-16
    author: Your Name
    approved_by: Head of Operations
    approval_date: 2026-09-16
    section: All
    change: First version
---

# Vendor Access Review

This note is the example that came with gdoc. Copy it, change the front matter
above and the words below, and you have a note of your own.

Everything between the two `---` lines is front matter. It is what goes on the
cover page and into the version control table, so the title here is the title
on the document. Everything after it is the body, written as ordinary markdown.

## What the review covers

Every vendor with access to a production system, and every account that access
is held under. The review runs every quarter and the owner named above signs it
off.

## How it runs

1. List every vendor account from the identity provider.
2. Ask the system owner whether each one is still needed.
3. Remove the accounts nobody claims.

The steps produce one record per vendor, kept with this note.

## What the record holds

| Column | Meaning |
| --- | --- |
| Vendor | The company the account belongs to |
| System | The system the account reaches |
| Owner | The person who confirmed the account is still needed |
| Reviewed | The date of the confirmation |

Try it:

```
gdoc build --md first-note.md --out first-note.docx
```

That writes the document on this machine and sends nothing anywhere. When it
looks right, `gdoc publish` puts it in Drive.
