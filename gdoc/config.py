"""Persistent settings that are not secrets."""

import json
from dataclasses import dataclass
from pathlib import Path

DEFAULT_PATH = Path.home() / ".config" / "gdoc-agent" / "config.json"


@dataclass(frozen=True)
class Config:
    output_folder_id: str | None = None


def load_config(path: Path | None = None) -> Config:
    """Read the config file.

    output_folder_id is optional, because generation falls back to a local .docx
    when there is no folder. Any other key in the file is ignored, so an older
    config that still carries display_name loads unchanged.
    """
    path = path or DEFAULT_PATH
    if not path.exists():
        raise FileNotFoundError(
            f"config not found at {path}. Create it with the Drive folder new "
            'versions go in, for example: {"output_folder_id": "0AFolderId"}'
        )
    data = json.loads(path.read_text())
    return Config(output_folder_id=data.get("output_folder_id"))
