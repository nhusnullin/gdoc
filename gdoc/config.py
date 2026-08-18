"""Persistent settings that are not secrets."""

import json
import os
import tempfile
from dataclasses import dataclass
from pathlib import Path

from gdoc.render import profiles

DEFAULT_PATH = Path.home() / ".config" / "gdoc-agent" / "config.json"

AUTH_MODES = ("oauth", "service_account")
DEFAULT_AUTH_MODE = "oauth"


@dataclass(frozen=True)
class Config:
    output_folder_id: str | None = None
    template: str = profiles.DEFAULT_TEMPLATE
    auth_mode: str | None = None


def load_config(path: Path | None = None) -> Config:
    """Read the config file.

    output_folder_id is optional, because generation falls back to a local .docx
    when there is no folder. Any other key in the file is ignored, so an older
    config that still carries display_name loads unchanged.

    template names the house style every new version is rendered through. It
    defaults to the bundled profile, so a config written before templates
    existed still publishes a house-styled document. Set it to "none" to keep
    the plain pandoc path.

    auth_mode picks the credential, and stays None when the file does not say.
    Defaulting it here would flip a working service_account install to oauth the
    moment it upgraded, because a config written before the key existed cannot
    mention it. What an unstated mode means depends on which credential files
    exist, and gdoc.auth.resolve_auth_mode is the one place that decides.
    """
    path = path or DEFAULT_PATH
    if not path.exists():
        raise FileNotFoundError(
            f"config not found at {path}. Create it with the Drive folder new "
            'versions go in, for example: {"output_folder_id": "0AFolderId"}'
        )
    data = _read_object(path)
    auth_mode = data.get("auth_mode") or None
    if auth_mode is not None and auth_mode not in AUTH_MODES:
        raise ValueError(
            f"auth_mode in {path} is {auth_mode!r}. "
            f"It must be one of: {', '.join(AUTH_MODES)}"
        )
    return Config(
        output_folder_id=data.get("output_folder_id"),
        template=data.get("template") or profiles.DEFAULT_TEMPLATE,
        auth_mode=auth_mode,
    )


def write_auth_mode(mode: str, path: Path | None = None) -> str | None:
    """Set auth_mode in the config file. Returns the mode it replaced, or None.

    `gdoc auth login` calls this, because a login that left the file alone would
    report success and change nothing: every other command reads auth_mode, so
    the old credential would still be the one in use. Editing a JSON file by
    hand is not a setup step a person should have to find.

    Every other key is carried over untouched, display_name included. The file
    holds the author's settings, and this function was asked about one of them.

    A file that cannot be parsed is left exactly as it is. Rewriting it would
    replace settings that could not be read with a file holding only this one
    key, which loses them for good.
    """
    if mode not in AUTH_MODES:
        raise ValueError(
            f"auth_mode {mode!r} is not one of: {', '.join(AUTH_MODES)}"
        )
    path = path or DEFAULT_PATH
    data = _read_object(path) if path.exists() else {}
    previous = data.get("auth_mode")
    _write_private(path, {**data, "auth_mode": mode})
    return previous


def _read_object(path: Path) -> dict:
    """The file's JSON object. Refuses anything else, by name.

    json.loads accepts any JSON value, so a file holding a list or a bare null
    parses and then has no .get. Left to itself that surfaces as an AttributeError
    from deep inside a command, naming nothing.
    """
    text = path.read_text()
    try:
        data = json.loads(text)
    except ValueError as error:
        raise ValueError(f"{path} is not valid JSON: {error}") from error
    if not isinstance(data, dict):
        raise ValueError(f"{path} does not hold a JSON object")
    return data


def _write_private(path: Path, data: dict) -> None:
    """Write the file in one step, private from the moment it exists.

    Written to a temporary file beside the target and then renamed, because
    truncating the target first means a full disk or a signal mid-write leaves an
    empty config. That loses settings which were readable a moment earlier, and
    the empty file cannot be parsed, so the next write refuses too. os.replace is
    atomic within a directory, so a reader sees the old file or the new one.

    0600 is set on the temporary file, so the target is never briefly readable.
    The directory holds the token and the service account key, so it matches.
    """
    path.parent.mkdir(parents=True, mode=0o700, exist_ok=True)
    handle, temporary = tempfile.mkstemp(dir=path.parent, prefix=".config-", suffix=".json")
    try:
        os.fchmod(handle, 0o600)
        with os.fdopen(handle, "w") as stream:
            json.dump(data, stream, indent=2)
            stream.write("\n")
        os.replace(temporary, path)
    except BaseException:
        Path(temporary).unlink(missing_ok=True)
        raise
