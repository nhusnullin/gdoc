"""Runtime version, read once from installed package metadata.

pyproject.toml is the one source of truth. Reading it here at runtime, instead
of duplicating the string in a second file, means the two can never disagree.
"""

from importlib.metadata import PackageNotFoundError, version

try:
    __version__ = version("gdoc")
except PackageNotFoundError:
    # Running from a checkout that was never `pip install -e`'d.
    __version__ = "0.0.0+unknown"


def _parse(raw: str) -> tuple[int, int, int]:
    core = raw.split("+", 1)[0].split("-", 1)[0]
    parts = core.split(".")
    if len(parts) != 3 or not all(part.isdigit() for part in parts):
        raise ValueError(f"not a MAJOR.MINOR.PATCH version: {raw!r}")
    return tuple(int(part) for part in parts)


def satisfies_minimum(current: str, minimum: str) -> bool:
    """True when `current` is at least `minimum`, compared as (major, minor, patch)."""
    return _parse(current) >= _parse(minimum)
