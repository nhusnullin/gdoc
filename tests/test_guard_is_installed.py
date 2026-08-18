"""The guard is worth nothing if a module can build its own client.

An allowlist, in the style of tests/test_no_external_programs.py. It fails if
`googleapiclient.discovery` is reached from a second module, and it fails just
as loudly if `auth.py` stops reaching it, because that would mean the client is
being built somewhere this test is not looking.

The check parses the import graph rather than grepping for `build(`. A grep for
the two literals was the first version and it was trivially evaded: a module
could write `from googleapiclient import discovery` and call
`discovery.build(...)` without either string appearing. Every route to `build`
goes through an import of that module, so that is what is counted.
"""

import ast
from pathlib import Path

PACKAGE = Path(__file__).resolve().parent.parent / "gdoc"

# gdoc/auth.py is the one place a Drive client is constructed, and the one
# place GuardedHttp is installed. Adding a name here means adding an unguarded
# client, so do not.
MAY_BUILD_A_CLIENT = {"auth.py"}


def _imports_discovery(tree: ast.AST) -> bool:
    for node in ast.walk(tree):
        if isinstance(node, ast.Import):
            if any(a.name.startswith("googleapiclient.discovery") for a in node.names):
                return True
        elif isinstance(node, ast.ImportFrom):
            module = node.module or ""
            if module == "googleapiclient.discovery":
                return True
            if module == "googleapiclient" and any(
                a.name == "discovery" for a in node.names
            ):
                return True
    return False


def _modules_reaching_discovery() -> set[str]:
    found = set()
    for path in PACKAGE.rglob("*.py"):
        tree = ast.parse(path.read_text(), filename=str(path))
        if _imports_discovery(tree):
            found.add(path.relative_to(PACKAGE).as_posix())
    return found


def test_only_auth_can_build_a_drive_client():
    assert _modules_reaching_discovery() == MAY_BUILD_A_CLIENT


def test_auth_installs_the_guard_on_every_client_it_builds():
    """Every build() call in auth.py passes http=, never credentials=.

    Passing credentials= would hand googleapiclient an unwrapped transport,
    and the two arguments cannot be given together, so this is the whole test.
    """
    tree = ast.parse((PACKAGE / "auth.py").read_text())
    calls = [
        node
        for node in ast.walk(tree)
        if isinstance(node, ast.Call)
        and isinstance(node.func, ast.Name)
        and node.func.id == "build"
    ]
    assert calls, "auth.py no longer builds a client. Move this test with it."
    for call in calls:
        keywords = {kw.arg for kw in call.keywords}
        assert "http" in keywords
        assert "credentials" not in keywords
