"""Credentials and the Drive client."""

from pathlib import Path

from google.oauth2 import service_account
from googleapiclient.discovery import build

DEFAULT_KEY_PATH = Path.home() / ".config" / "gdoc-agent" / "sa-key.json"
SCOPES = ["https://www.googleapis.com/auth/drive"]
SPEC = "docs/superpowers/specs/2026-08-13-gdoc-ai-agent-design.md"


def load_credentials(key_path: Path | None = None):
    key_path = key_path or DEFAULT_KEY_PATH
    if not key_path.exists():
        raise FileNotFoundError(
            f"service account key not found at {key_path}. "
            f"Setup steps are in {SPEC}, section 10."
        )
    return service_account.Credentials.from_service_account_file(
        str(key_path), scopes=SCOPES
    )


def drive_service(credentials=None):
    """Build a Drive v3 client.

    cache_discovery is off because the on-disk discovery cache warns noisily
    under a venv and buys nothing for a tool that runs for a few seconds.
    """
    return build(
        "drive", "v3", credentials=credentials or load_credentials(), cache_discovery=False
    )
