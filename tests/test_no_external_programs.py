"""The publish path must not call out to anything, except the one thing it still does.

gdoc runs where the source documents live, and that is an environment with no
package manager. A dependency that cannot be pip installed cannot be installed at
all, so this is a rule rather than a preference.

Scoped to gdoc/render/, the publish path. gdoc/export.py still shells out to pandoc
as a markdown fallback, and gdoc/generate.py keeps a plain pandoc path for
`--template none`. Both are out of scope here: Drive exports text/markdown
natively, so the fallback is removable, and the plain path is a deliberate
fallback rather than a leftover. Widen this guard if either goes.

pandoc is also, today, the one external program still used inside gdoc/render/
itself: gdoc/render/body.py runs it to parse markdown into an AST, and never as a
docx writer. That is why pandoc is not in PROGRAMS below, and why the subprocess
check is an allowlist of exactly one file rather than a blanket ban. Removing it
means swapping in a pure-Python markdown parser and rewriting body.py's AST walker,
which is most of the file and is where the document body's pixel fidelity lives.
That is a separate change with its own spec, not a step in this plan.
"""

import ast
from pathlib import Path

import pytest

RENDER = Path(__file__).resolve().parent.parent / "gdoc" / "render"
PROGRAMS = ("soffice", "libreoffice", "pdftotext", "pdfinfo", "pdftoppm")


def python_files():
    return sorted(RENDER.rglob("*.py"))


def _docstring_owners(tree):
    """The module itself, plus every class and function in it."""
    return [tree] + [
        node for node in ast.walk(tree)
        if isinstance(node, (ast.ClassDef, ast.FunctionDef, ast.AsyncFunctionDef))
    ]


def code_only(text):
    """Source with real docstrings and comments stripped, real code kept.

    Blanks only the line spans of genuine docstrings, found with
    `ast.get_docstring` on the module and on every class and function, never by
    matching triple quotes. A triple-quoted string used as an ordinary value,
    such as a command template assigned to a variable, is not a docstring by
    this test's own admission and survives untouched, same as it would survive
    at runtime.

    Naming a program while explaining why it is gone is fine, and several
    modules do exactly that. Only executable code counts.
    """
    lines = text.splitlines()
    tree = ast.parse(text)
    for owner in _docstring_owners(tree):
        if ast.get_docstring(owner) is None:
            continue
        docstring_statement = owner.body[0]
        for line_number in range(docstring_statement.lineno, docstring_statement.end_lineno + 1):
            lines[line_number - 1] = ""
    without_docstrings = "\n".join(lines)
    return "\n".join(
        line for line in without_docstrings.splitlines()
        if not line.strip().startswith("#")
    )


def test_there_are_python_files_to_check():
    """Guards the guard. An empty glob would make every check below vacuous."""
    assert len(python_files()) > 3


def test_the_stripper_keeps_code_and_drops_prose():
    """Guards the guard again. A stripper that ate everything would pass silently.

    Also guards against the shape of bug that broke an earlier, regex-based
    version of this stripper: a triple-quoted string assigned to a variable is
    live code, not a docstring, whatever quote style it uses, and must survive
    stripping exactly like the double-quoted assignment below does.
    """
    sample = (
        '"""a docstring naming soffice"""\n'
        '# a comment naming pandoc\n'
        'PDFTOTEXT_CMD = """a live triple-quoted string naming pdftotext"""\n'
        'x = "kept"\n'
    )
    stripped = code_only(sample)
    assert "soffice" not in stripped
    assert "pandoc" not in stripped
    assert "pdftotext" in stripped
    assert "kept" in stripped


def test_only_body_imports_subprocess():
    """pandoc is the one external program the publish path still uses.

    gdoc/render/body.py runs it to parse markdown into an AST, and only as a
    parser: it is never used to write the docx. Removing it needs a pure-Python
    markdown parser and a rewrite of body.py's AST walker, which is a separate
    change with its own spec.

    This is an allowlist, not a ban, and that is deliberate: it fails if
    subprocess spreads to a second module, and it fails just as loudly if
    body.py's own subprocess import disappears without this test being updated,
    since a stale allowlist would silently stop proving anything.
    """
    importers = set()
    for path in python_files():
        tree = ast.parse(path.read_text(encoding="utf-8"))
        for node in ast.walk(tree):
            if isinstance(node, ast.Import):
                if any(alias.name.split(".")[0] == "subprocess" for alias in node.names):
                    importers.add(path.name)
            elif isinstance(node, ast.ImportFrom):
                if (node.module or "").split(".")[0] == "subprocess":
                    importers.add(path.name)
    assert importers == {"body.py"}, f"subprocess is imported by {sorted(importers)}"


@pytest.mark.parametrize("program", PROGRAMS)
def test_no_render_module_names_an_external_program_in_code(program):
    offenders = [
        path.name for path in python_files()
        if program in code_only(path.read_text(encoding="utf-8"))
    ]
    assert not offenders, f"{program} appears in the code of {offenders}"


def test_no_shell_scripts_ship_inside_the_package():
    package = RENDER.parent
    scripts = list(package.rglob("*.sh")) + list(package.rglob("*.bas"))
    assert not scripts, f"external scripts still shipped: {scripts}"
