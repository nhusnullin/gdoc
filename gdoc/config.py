"""Persistent settings that are not secrets."""

import json
from dataclasses import dataclass
from pathlib import Path

DEFAULT_PATH = Path.home() / ".config" / "gdoc-agent" / "config.json"


@dataclass(frozen=True)
class Config:
    display_name: str
    output_folder_id: str | None = None


def load_config(path: Path | None = None) -> Config:
    """Read the config file.

    display_name is required because it is the only way to tell Nail's comments
    from a colleague's: the Drive API returns no email address for comment
    authors. output_folder_id is optional, because generation falls back to a
    local .docx when there is no folder.
    """
    path = path or DEFAULT_PATH
    if not path.exists():
        raise FileNotFoundError(
            f"config not found at {path}. Create it with a display_name, "
            'for example: {"display_name": "Nail Khusnullin", "output_folder_id": null}'
        )
    data = json.loads(path.read_text())
    display_name = data.get("display_name")
    if not display_name:
        raise ValueError(f"display_name is required in {path}")
    return Config(
        display_name=display_name,
        output_folder_id=data.get("output_folder_id"),
    )
