"""The user credential, tested without a browser and without the network.

Real google.oauth2.credentials.Credentials objects are used throughout, because
they parse and serialise the token file the code actually writes. Only refresh is
patched, and it is patched for every test in the file, because of the trap below.

The trap, verified against google-auth 2.56.3 on 2026-08-14: when a token file
has no `expiry` key, from_authorized_user_info stamps expiry with the CURRENT
time, so the token loads as already expired. A real token from the browser flow
always has an expiry, so this only bites hand written files and test fixtures.
Get it wrong in a test and the test reaches Google.

Expiry values must be naive UTC. google-auth compares expiry against its own
naive _helpers.utcnow(), and mixing an aware datetime in raises TypeError.
"""

import json
import stat
from datetime import datetime, timedelta, timezone

import pytest
from google.oauth2.credentials import Credentials

from gdoc import oauth

SCOPES = ["https://www.googleapis.com/auth/drive"]
READONLY = "https://www.googleapis.com/auth/drive.readonly"


def naive_utc(delta=timedelta()):
    """Naive UTC, because that is what google-auth compares expiry against."""
    return datetime.now(timezone.utc).replace(tzinfo=None) + delta


@pytest.fixture(autouse=True)
def never_reach_google(monkeypatch):
    """No test in this file may leave the machine.

    load() refreshes whenever the token is not valid, and a fixture without a
    future expiry is not valid. Patching this for every test is the only way to
    be sure a new test cannot quietly start calling Google.
    """

    def fake_refresh(self, request):
        self.token = "a-fresh-token"
        self.expiry = naive_utc(timedelta(hours=1))

    monkeypatch.setattr(Credentials, "refresh", fake_refresh)


def credentials(expiry=None, scopes=SCOPES):
    return Credentials(
        token="an-access-token",
        refresh_token="a-refresh-token",
        token_uri="https://oauth2.googleapis.com/token",
        client_id="client-id",
        client_secret="client-secret",
        scopes=scopes,
        expiry=expiry if expiry is not None else naive_utc(timedelta(hours=1)),
    )


def write_token(path, creds):
    path.write_text(creds.to_json())
    return path


# --- load ---------------------------------------------------------------


def test_a_valid_token_loads_without_refreshing(tmp_path):
    token = write_token(tmp_path / "token.json", credentials())
    loaded = oauth.load(SCOPES, token_path=token)
    assert loaded.valid is True
    assert loaded.token == "an-access-token"


def test_a_token_file_with_no_expiry_is_refreshed_on_load(tmp_path):
    """Documents the trap in the module docstring.

    google-auth stamps a missing expiry with the current time, so such a file is
    expired the moment it is read. Refreshing is the right answer, and the point
    of the test is that the behaviour is known rather than discovered.
    """
    token = tmp_path / "token.json"
    token.write_text(
        json.dumps(
            {
                "token": "an-access-token",
                "refresh_token": "a-refresh-token",
                "client_id": "c",
                "client_secret": "s",
                "token_uri": "https://oauth2.googleapis.com/token",
                "scopes": SCOPES,
            }
        )
    )
    assert oauth.load(SCOPES, token_path=token).token == "a-fresh-token"


def test_a_missing_token_names_the_path_and_both_ways_out(tmp_path):
    token = tmp_path / "token.json"
    with pytest.raises(FileNotFoundError) as excinfo:
        oauth.load(SCOPES, token_path=token)
    message = str(excinfo.value)
    assert str(token) in message
    assert "gdoc auth login" in message
    assert "service_account" in message


def test_an_expired_token_refreshes_and_the_file_is_rewritten(tmp_path):
    token = write_token(
        tmp_path / "token.json", credentials(expiry=naive_utc(timedelta(hours=-2)))
    )
    loaded = oauth.load(SCOPES, token_path=token)
    assert loaded.token == "a-fresh-token"
    assert json.loads(token.read_text())["token"] == "a-fresh-token"


def test_the_rewritten_token_file_is_still_private(tmp_path):
    token = write_token(
        tmp_path / "token.json", credentials(expiry=naive_utc(timedelta(hours=-2)))
    )
    token.chmod(0o644)
    oauth.load(SCOPES, token_path=token)
    assert stat.S_IMODE(token.stat().st_mode) == 0o600


def test_a_token_missing_scopes_is_refused_at_load(tmp_path):
    token = write_token(tmp_path / "token.json", credentials(scopes=[READONLY]))
    with pytest.raises(PermissionError) as excinfo:
        oauth.load(SCOPES, token_path=token)
    message = str(excinfo.value)
    assert "auth/drive" in message
    assert "gdoc auth login" in message


def test_the_scope_check_reads_the_file_not_the_object(tmp_path):
    """from_authorized_user_file reports the scopes it was ASKED for.

    So the object always looks correctly scoped, and only the file tells the
    truth. If this test ever passes while the previous one fails, the check moved
    to the object and stopped working.
    """
    token = write_token(tmp_path / "token.json", credentials(scopes=[READONLY]))
    loaded = Credentials.from_authorized_user_file(str(token), SCOPES)
    assert loaded.scopes == SCOPES


def test_a_token_file_that_cannot_be_used_says_so_rather_than_crashing(tmp_path):
    token = tmp_path / "token.json"
    token.write_text(json.dumps({"token": "t"}))
    with pytest.raises(PermissionError) as excinfo:
        oauth.load(SCOPES, token_path=token)
    assert "gdoc auth login" in str(excinfo.value)


def test_a_token_file_with_no_scopes_recorded_is_not_refused(tmp_path):
    """Only a hand written file lacks the key. Drive will say if it is wrong.

    Refusing on unknown would force a needless re-login. to_json always writes
    the key, so this only covers a file someone typed.
    """
    token = tmp_path / "token.json"
    token.write_text(
        json.dumps(
            {
                "token": "t",
                "refresh_token": "r",
                "client_id": "c",
                "client_secret": "s",
                "token_uri": "https://oauth2.googleapis.com/token",
                "expiry": naive_utc(timedelta(hours=1)).isoformat(),
            }
        )
    )
    assert oauth.load(SCOPES, token_path=token).token == "t"


# --- login --------------------------------------------------------------


def test_login_without_a_client_file_names_the_path_and_the_spec(tmp_path):
    client = tmp_path / "oauth-client.json"
    with pytest.raises(FileNotFoundError) as excinfo:
        oauth.login(SCOPES, client_path=client, token_path=tmp_path / "token.json")
    message = str(excinfo.value)
    assert str(client) in message
    assert "Desktop" in message
    assert "2026-08-14-gdoc-oauth-design" in message


def test_login_writes_the_token_private(tmp_path, monkeypatch):
    client = tmp_path / "oauth-client.json"
    client.write_text(json.dumps({"installed": {"client_id": "c", "client_secret": "s"}}))
    token = tmp_path / "token.json"

    class FakeFlow:
        @classmethod
        def from_client_secrets_file(cls, path, scopes):
            assert scopes == SCOPES
            return cls()

        def run_local_server(self, **kwargs):
            assert kwargs["access_type"] == "offline"
            assert kwargs["prompt"] == "consent"
            return credentials()

    monkeypatch.setattr(oauth, "InstalledAppFlow", FakeFlow)

    oauth.login(SCOPES, client_path=client, token_path=token)
    assert token.exists()
    assert stat.S_IMODE(token.stat().st_mode) == 0o600


# --- logout -------------------------------------------------------------


def test_logout_removes_the_token_and_says_it_did(tmp_path):
    token = write_token(tmp_path / "token.json", credentials())
    assert oauth.logout(token_path=token) is True
    assert not token.exists()


def test_logout_on_a_machine_that_never_logged_in_says_so(tmp_path):
    assert oauth.logout(token_path=tmp_path / "token.json") is False


# --- account ------------------------------------------------------------


def test_account_asks_drive_who_is_signed_in():
    from unittest.mock import MagicMock

    drive = MagicMock()
    drive.about().get.return_value.execute.return_value = {
        "user": {"displayName": "Nail Khusnullin", "emailAddress": "nail@altery.com"}
    }
    assert oauth.account(drive)["emailAddress"] == "nail@altery.com"


def test_account_survives_a_reply_with_no_user_block():
    from unittest.mock import MagicMock

    drive = MagicMock()
    drive.about().get.return_value.execute.return_value = {}
    assert oauth.account(drive) == {}
