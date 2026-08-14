"""Resolve a template choice to the master .docx it names.

A profile is a directory holding template.docx. Bundled profiles live in
gdoc/templates/, and the choice is a config value rather than a constant, so
nothing in the code branches on which house style is in use.

There is no plugin API for the surgery, deliberately. shell.py finds the cover
by placeholder text and the tables by their first-column labels, so a .docx that
does not follow that contract fails there, with a message naming what was not
found. When a second real template arrives, this is where a per-profile surgery
module hooks in.
"""

from pathlib import Path

TEMPLATES_DIR = Path(__file__).resolve().parent.parent / "templates"
DEFAULT_TEMPLATE = "altery-group-policy-v1.0"
MASTER_NAME = "template.docx"
NO_TEMPLATE = "none"


class ProfileError(RuntimeError):
    """The named template does not exist, or does not hold a master."""


def available() -> list[str]:
    if not TEMPLATES_DIR.is_dir():
        return []
    return sorted(p.name for p in TEMPLATES_DIR.iterdir() if (p / MASTER_NAME).is_file())


def resolve(spec: str) -> Path | None:
    """Return the master .docx for this spec, or None for the plain pandoc path.

    A spec that exists on disk is a path, either to a .docx or to a profile
    directory. Anything else is the name of a bundled profile.
    """
    if spec == NO_TEMPLATE:
        return None
    candidate = Path(spec).expanduser()
    if candidate.is_file():
        return candidate
    if candidate.is_dir():
        master = candidate / MASTER_NAME
        if not master.is_file():
            raise ProfileError(f"{candidate} holds no {MASTER_NAME}")
        return master
    master = TEMPLATES_DIR / spec / MASTER_NAME
    if not master.is_file():
        raise ProfileError(
            f"unknown template {spec!r}. Bundled templates: {available()}. "
            "A path to a .docx or to a profile directory also works, "
            f"and {NO_TEMPLATE!r} means no template at all."
        )
    return master
