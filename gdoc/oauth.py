"""The user credential.

A service account needs every document shared with its address first. OAuth does
not: the agent authorises once as Nail and reaches whatever Nail can reach.

The trade is the full Drive scope, which Google classes as restricted. Not because
comments need it: comments.list and replies.create both accept drive.file, which is
non-sensitive. drive.file covers only files the app created or that the user handed
it through the Google Picker or Drive's "Open with", and there is no way to put an
existing file in that scope by pasting a URL, which is the only gesture a terminal
has. So gdoc/guard.py carries the cannot-edit promise instead.

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

# The OAuth client gdoc signs in with, so nobody using it has to visit a cloud
# console first. Safe to keep in version control, and not because this repo is
# private: RFC 8252 section 8.5 says a secret shipped to many users "should not
# be treated as confidential", serves no purpose "beyond client identification",
# and the server MUST treat such a client as public. What protects an account is
# the per-user token in oauth-token.json, which never leaves the machine.
#
# gh ships its client id and secret the same way, with the comment "This value
# is safe to be embedded in version control". gcloud ships a Google client
# secret in a constant it named CLOUDSDK_CLIENT_NOTSOSECRET.
#
# The client must stay User type Internal. That exempts gdoc from OAuth
# verification, from the unverified-app screen and from the 100-user cap, which
# matters because the Drive scope it needs is a restricted scope. Going External
# would mean a CASA security assessment every 12 months.
#
# The cost of sharing one client is quota, not security: Google rate limits per
# client, and every user of this one draws on the same allowance. rclone is
# retiring its shared Drive client during 2026 for exactly that reason, at a
# scale of many hundreds of times the free quota. Anyone who needs their own can
# drop a client file in place, which wins over this one.
BUNDLED_CLIENT_ID = ""
BUNDLED_CLIENT_SECRET = ""

AUTH_URI = "https://accounts.google.com/o/oauth2/auth"
TOKEN_URI = "https://oauth2.googleapis.com/token"

LOGIN = "Run: gdoc auth login"
USE_THE_KEY_INSTEAD = (
    "To use the service account instead, run: gdoc auth use service_account. "
    "It writes the setting, so there is no file to edit by hand."
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

    Anything unreadable answers None, which means "the file does not say" and
    leaves the refusal to the parse below. json.loads succeeds on any JSON
    value, so a file holding a list or a bare null parses fine and then has no
    .get; and the path may be unreadable, or a directory. None of those may
    escape as an AttributeError or an OSError, because the caller's message is
    the one that names `gdoc auth login`.
    """
    try:
        data = json.loads(path.read_text())
    except (ValueError, OSError):
        return None
    if not isinstance(data, dict):
        return None
    scopes = data.get("scopes")
    return set(scopes) if isinstance(scopes, list) else None


def client_config(client_path=None) -> tuple[dict, str]:
    """The OAuth client to sign in with, and where it came from.

    A client file wins over the bundled one. The reason to bring your own is
    quota rather than security, so it is the owner's call and never a setup step.
    gcloud treats --client-id-file the same way.

    Both halves of the bundled client have to be there. An id with no secret
    would fail at Google with a message about the request rather than about the
    build.
    """
    client_path = Path(client_path or DEFAULT_CLIENT_PATH)
    if client_path.exists():
        return json.loads(client_path.read_text()), str(client_path)
    if BUNDLED_CLIENT_ID and BUNDLED_CLIENT_SECRET:
        return (
            {
                "installed": {
                    "client_id": BUNDLED_CLIENT_ID,
                    "client_secret": BUNDLED_CLIENT_SECRET,
                    "auth_uri": AUTH_URI,
                    "token_uri": TOKEN_URI,
                }
            },
            "bundled",
        )
    raise FileNotFoundError(
        f"this build ships no OAuth client, and there is none at {client_path}. "
        f"Create a Desktop app OAuth client in Google Cloud, User type Internal, "
        f"and save its JSON there. Steps are in {SPEC}, section 9."
    )


def login(scopes, client_path=None, token_path=None) -> Credentials:
    """Run the browser flow and save the result."""
    token_path = Path(token_path or DEFAULT_TOKEN_PATH)
    config, _ = client_config(client_path)
    flow = InstalledAppFlow.from_client_config(config, list(scopes))
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
    except (ValueError, AttributeError, TypeError, OSError) as error:
        # Everything a bad token file can throw, turned into the one message
        # that names the fix. json.loads accepts any JSON value, so a file
        # holding a list or a bare null gets past the parse and then fails on
        # attribute access; and the path may be unreadable or a directory.
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
