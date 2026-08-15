"""Credentials and the Drive client.

One chokepoint, so nothing else in the package has to know which credential is
in use. Two modes:

- service_account, shared onto a document as Commenter. Google itself refuses
  every edit, and the credential reaches only what has been shared with it.
- oauth, authorised as Nail. No sharing step, but the token is Nail and the
  scope is full Drive, so the reachable set is everything Nail owns. There is
  no narrower scope that reads comments and writes replies, so gdoc/guard.py
  narrows it here instead.

The guard is installed in both modes and cannot be switched off. Under the
service account it is nearly redundant, and one code path is worth more than a
saved wrapper.

doc_ids is the set of files a command may touch. It is empty for commands that
have no input document, such as generate, which creates its own and lets the
guard learn the ids from the create responses.
"""

from pathlib import Path

import google_auth_httplib2
import httplib2
from google.oauth2 import service_account
from googleapiclient.discovery import build

from gdoc import oauth
from gdoc.config import DEFAULT_AUTH_MODE, Config, load_config
from gdoc.guard import GuardedHttp

DEFAULT_KEY_PATH = Path.home() / ".config" / "gdoc-agent" / "sa-key.json"
SCOPES = ["https://www.googleapis.com/auth/drive"]
SPEC = "docs/superpowers/specs/2026-08-13-gdoc-ai-agent-design.md"


def _config():
    """The config, or its defaults.

    A missing config.json is not a failure. gdoc read works without one, and
    every key this module reads has a documented default.
    """
    try:
        return load_config()
    except (FileNotFoundError, ValueError):
        return Config()


def load_service_account_credentials(key_path: Path | None = None):
    key_path = key_path or DEFAULT_KEY_PATH
    if not key_path.exists():
        raise FileNotFoundError(
            f"service account key not found at {key_path}. "
            f"Setup steps are in {SPEC}, section 10."
        )
    return service_account.Credentials.from_service_account_file(
        str(key_path), scopes=SCOPES
    )


def load_credentials(key_path: Path | None = None, mode: str | None = None):
    """Pick a credential. The one place that decides."""
    mode = mode or _config().auth_mode or DEFAULT_AUTH_MODE
    if mode == "oauth":
        return oauth.load(SCOPES)
    return load_service_account_credentials(key_path)


def drive_service(credentials=None, doc_ids=()):
    """Build a Drive v3 client that reaches only doc_ids.

    http is passed instead of credentials, because the guard has to wrap the
    transport and googleapiclient refuses both arguments at once.

    A bare string is accepted, because every caller but generate has exactly
    one document and should not have to wrap it.

    cache_discovery is off because the on-disk discovery cache warns noisily
    under a venv and buys nothing for a tool that runs for a few seconds.
    """
    if isinstance(doc_ids, str):
        doc_ids = (doc_ids,)
    credentials = credentials or load_credentials()
    http = google_auth_httplib2.AuthorizedHttp(credentials, http=httplib2.Http())
    return build(
        "drive", "v3", http=GuardedHttp(http, doc_ids), cache_discovery=False
    )
