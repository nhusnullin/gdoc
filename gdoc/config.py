"""Persistent settings that are not secrets."""

import json
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
    auth_mode: str = DEFAULT_AUTH_MODE


def load_config(path: Path | None = None) -> Config:
    """Read the config file.

    output_folder_id is optional, because generation falls back to a local .docx
    when there is no folder. Any other key in the file is ignored, so an older
    config that still carries display_name loads unchanged.

    template names the house style every new version is rendered through. It
    defaults to the bundled profile, so a config written before templates
    existed still publishes a house-styled document. Set it to "none" to keep
    the plain pandoc path.

    auth_mode picks the credential and defaults to oauth, so a config written
    before the key existed keeps loading.
    """
    path = path or DEFAULT_PATH
    if not path.exists():
        raise FileNotFoundError(
            f"config not found at {path}. Create it with the Drive folder new "
            'versions go in, for example: {"output_folder_id": "0AFolderId"}'
        )
    data = json.loads(path.read_text())
    auth_mode = data.get("auth_mode") or DEFAULT_AUTH_MODE
    if auth_mode not in AUTH_MODES:
        raise ValueError(
            f"auth_mode in {path} is {auth_mode!r}. "
            f"It must be one of: {', '.join(AUTH_MODES)}"
        )
    return Config(
        output_folder_id=data.get("output_folder_id"),
        template=data.get("template") or profiles.DEFAULT_TEMPLATE,
        auth_mode=auth_mode,
    )
