"""The guard is worth nothing if a module can build its own client.

An allowlist, in the style of tests/test_no_external_programs.py. It fails if
the call to build() spreads to a second module, and it fails just as loudly if
auth.py stops making it, because that would mean the client is being built
somewhere this test is not looking.
"""

from pathlib import Path

PACKAGE = Path(__file__).resolve().parent.parent / "gdoc"


def _modules_calling_build():
    found = set()
    for path in PACKAGE.rglob("*.py"):
        if "build(" in path.read_text() and "discovery import build" in path.read_text():
            found.add(path.relative_to(PACKAGE).as_posix())
    return found


def test_only_auth_builds_the_drive_client():
    assert _modules_calling_build() == {"auth.py"}
