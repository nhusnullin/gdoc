---
worth: later
where: release/install.sh:498
added: 2026-10-03
---
# The installer's quit-and-reopen step is easy to miss

An extension installed into a Claude Desktop that is already running starts,
and the bridge announces its tools, but no chat can see them until Claude
Desktop is quit and opened again. Chat then says it has no gdoc tools and gives
no reason. Nail hit this in the clean install on 2026-10-03 (MEASURED.md, "the
red team and a clean install").

The installer already says it, but in the middle of its other output:

    opened. Claude Desktop asks to install it, then quit it and open it again.

The fix is to make that step hard to miss: a separate line at the very end of
the output, for example

    NEXT: quit Claude Desktop (⌘Q) and open it again. Until then chat cannot see gdoc.

The same applies to `gdoc update --desktop`, which ends the same way.

Worth `later` because chat in Claude Desktop ships silent in v2.9.0, so no
colleague installs the extension yet. It becomes `yes` when Nail announces chat.
