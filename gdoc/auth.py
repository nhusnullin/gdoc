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
from gdoc.config import AUTH_MODES, DEFAULT_AUTH_MODE, Config, load_config
from gdoc.guard import GuardedHttp

DEFAULT_KEY_PATH = Path.home() / ".config" / "gdoc-agent" / "sa-key.json"
SCOPES = ["https://www.googleapis.com/auth/drive"]
SPEC = "docs/superpowers/specs/2026-08-13-gdoc-ai-agent-design.md"


def _config():
    """The config, or its defaults when there is no file.

    A missing config.json is not a failure. gdoc read works without one, and
    every key this module reads has a documented default.

    A config that exists but cannot be understood is a different thing, and it
    reaches the caller. Swallowing it here meant a typo in auth_mode fell through
    to inference, and on a machine holding a token that resolved to oauth, the
    credential with full Drive scope, when the author had asked for the narrow
    one. Not knowing must never resolve to the wider answer.
    """
    try:
        return load_config()
    except FileNotFoundError:
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


def configured_mode() -> str | None:
    """The mode the config states, or None when it does not say.

    Separate from the resolver, which is then a pure function of what it is
    handed. That is what lets a caller ask the two questions apart: what did the
    author state, and what does that resolve to.
    """
    return _config().auth_mode


def resolve_auth_mode(
    mode: str | None,
    key_path: Path | None = None,
    token_path: Path | None = None,
) -> str:
    """Which credential to use. The one place that decides.

    A mode stated in the config wins outright. It is the author's word, and a
    file appearing in ~/.config/gdoc-agent must never overrule it.

    An unstated mode is inferred from what is installed, because the config
    cannot mention a key that did not exist when it was written:

    - a token, so somebody signed in. Signing in is the deliberate act, so it
      decides even when a key is also present.
    - a key and no token, so this is a service_account install that predates
      oauth. Defaulting it to oauth would break a setup that works today.
    - neither, so this is a new install and oauth is the default. It needs no
      sharing step, which is the whole reason it is the default.

    mode is what the config states, and None means it stated nothing. This
    function never reads the config, so the two cases stay distinguishable. A
    mode it does not recognise is refused rather than inferred past, for the
    reason in _config.

    The paths are read through the module globals rather than captured at import,
    so a test can point them somewhere and the suite stops depending on which
    credentials the developer happens to have.
    """
    if mode is not None and mode not in AUTH_MODES:
        raise ValueError(
            f"auth_mode {mode!r} is not one of: {', '.join(AUTH_MODES)}"
        )
    if mode is not None:
        return mode
    key_path = Path(key_path or DEFAULT_KEY_PATH)
    token_path = Path(token_path or oauth.DEFAULT_TOKEN_PATH)
    if token_path.exists():
        return "oauth"
    if key_path.exists():
        return "service_account"
    return DEFAULT_AUTH_MODE


def load_credentials(key_path: Path | None = None, mode: str | None = None):
    """Read the credential the resolver picked."""
    mode = resolve_auth_mode(mode or configured_mode(), key_path=key_path)
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
