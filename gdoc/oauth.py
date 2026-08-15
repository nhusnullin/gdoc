"""The user credential.

A service account needs every document shared with its address first. OAuth does
not: the agent authorises once as Nail and reaches whatever Nail can reach. The
trade is that Drive has no scope which reads comments and writes replies without
full Drive access, so gdoc/guard.py carries the cannot-edit promise instead.

This module takes scopes as an argument and never imports gdoc.auth, so the
chokepoint can depend on this without a cycle.
"""

import json
import os
from pathlib import Path

from google.auth.transport.requests import Request
from google.oauth2.credentials import Credentials
from google_auth_oauthlib.flow import InstalledAppFlow

CONFIG_DIR = Path.home() / ".config" / "gdoc-agent"
DEFAULT_CLIENT_PATH = CONFIG_DIR / "oauth-client.json"
DEFAULT_TOKEN_PATH = CONFIG_DIR / "oauth-token.json"
SPEC = "docs/superpowers/specs/2026-08-14-gdoc-oauth-design.md"

LOGIN = "Run: gdoc auth login"
USE_THE_KEY_INSTEAD = (
    'To use the service account instead, set "auth_mode": "service_account" '
    "in ~/.config/gdoc-agent/config.json."
)


def _write_token(path: Path, credentials: Credentials) -> None:
    """Create the file private, then fill it. Never the other way round.

    The chmod is not redundant. O_CREAT applies its mode only when the file does
    not already exist, so refreshing into a file someone had loosened would leave
    it loosened. This is the one file in the tool that holds a live credential.
    """
    path.parent.mkdir(parents=True, exist_ok=True)
    handle = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0o600)
    with os.fdopen(handle, "w") as stream:
        stream.write(credentials.to_json())
    os.chmod(path, 0o600)


def _stored_scopes(path: Path) -> set[str] | None:
    """The scopes the token records, or None when the file does not say.

    Read from the file rather than the Credentials object, because
    from_authorized_user_file reports back the scopes it was asked for, not the
    ones the token actually carries.
    """
    try:
        data = json.loads(path.read_text())
    except ValueError:
        return None
    scopes = data.get("scopes")
    return set(scopes) if isinstance(scopes, list) else None


def login(scopes, client_path=None, token_path=None) -> Credentials:
    """Run the browser flow and save the result."""
    client_path = Path(client_path or DEFAULT_CLIENT_PATH)
    token_path = Path(token_path or DEFAULT_TOKEN_PATH)
    if not client_path.exists():
        raise FileNotFoundError(
            f"OAuth client not found at {client_path}. Create a Desktop app "
            f"OAuth client in Google Cloud and save its JSON there. "
            f"Steps are in {SPEC}, section 9."
        )
    flow = InstalledAppFlow.from_client_secrets_file(str(client_path), list(scopes))
    # Both keyword arguments are passed rather than relied on. Without them
    # Google may return no refresh token, and the next run would silently need a
    # browser again.
    credentials = flow.run_local_server(
        port=0, access_type="offline", prompt="consent"
    )
    _write_token(token_path, credentials)
    return credentials


def load(scopes, token_path=None) -> Credentials:
    """Read the saved token, refreshing it if it has expired.

    A token file with no expiry recorded counts as expired, because google-auth
    stamps a missing expiry with the current time. That is fine: it refreshes and
    the file is rewritten with a real expiry. Worth knowing before you conclude
    the refresh path is running when it should not be.
    """
    token_path = Path(token_path or DEFAULT_TOKEN_PATH)
    if not token_path.exists():
        raise FileNotFoundError(
            f"no OAuth token at {token_path}. {LOGIN}\n{USE_THE_KEY_INSTEAD}"
        )

    stored = _stored_scopes(token_path)
    if stored is not None:
        missing = set(scopes) - stored
        if missing:
            raise PermissionError(
                f"the token at {token_path} is missing "
                f"{', '.join(sorted(missing))}. {LOGIN}"
            )

    try:
        credentials = Credentials.from_authorized_user_file(
            str(token_path), list(scopes)
        )
    except ValueError as error:
        raise PermissionError(
            f"the token at {token_path} cannot be used ({error}). {LOGIN}"
        ) from error

    if credentials.valid:
        return credentials

    # A token file always carries a refresh token, because
    # from_authorized_user_file refuses to parse one without it.
    credentials.refresh(Request())
    _write_token(token_path, credentials)
    return credentials


def logout(token_path=None) -> bool:
    """Delete the local token. Returns whether there was one.

    The grant with Google is untouched. Revoking it is a browser action, and
    gdoc auth status prints where.
    """
    token_path = Path(token_path or DEFAULT_TOKEN_PATH)
    if not token_path.exists():
        return False
    token_path.unlink()
    return True


def account(drive) -> dict:
    """Who the credential is.

    Asked of Drive, so knowing the signed-in account costs no extra scope.
    """
    about = drive.about().get(fields="user(displayName,emailAddress)").execute()
    return about.get("user") or {}
