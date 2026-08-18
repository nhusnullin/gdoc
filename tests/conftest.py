import pytest

TEST_DOC_ID = "1tyVhOTw9-bJ99IJTBZoTfWAT6fm3rjGIzfpHQX91hkw"

# A real Drive id shape that is not the test document. Every request to it must
# be refused inside the process, so it is never fetched and never has to exist.
OTHER_DOC_ID = "1NotTheDocumentTheseTestsWereGiven00000000"


@pytest.fixture(autouse=True)
def gdoc_agent_dir(request, tmp_path_factory, monkeypatch):
    """Every unit test gets its own empty ~/.config/gdoc-agent. Returns the path.

    Autouse, because `gdoc auth login` writes auth_mode into the config file. A
    test that forgot to redirect the path would edit the developer's real config,
    and a test suite must not have side effects outside its tmp directory. This
    is not politeness: the file decides which credential every command uses.

    Resolving an unstated auth_mode also reads which credential files exist, so
    an empty directory is what keeps that answer the same on every machine.

    Integration tests are exempt. They call Drive, so the real credential is the
    point of them.
    """
    if request.node.get_closest_marker("integration"):
        yield None
        return
    home = tmp_path_factory.mktemp("gdoc-agent")
    monkeypatch.setattr("gdoc.config.DEFAULT_PATH", home / "config.json")
    monkeypatch.setattr("gdoc.auth.DEFAULT_KEY_PATH", home / "sa-key.json")
    monkeypatch.setattr("gdoc.oauth.DEFAULT_TOKEN_PATH", home / "oauth-token.json")
    # All four paths, not the three the tool writes. A test that forgot to patch
    # oauth.login would otherwise read the developer's real client and open a
    # browser mid-suite.
    monkeypatch.setattr("gdoc.oauth.DEFAULT_CLIENT_PATH", home / "oauth-client.json")
    yield home


@pytest.fixture(scope="session")
def auth_mode():
    """The configured mode, or the default when there is no config."""
    from gdoc.config import Config, load_config

    from gdoc.auth import resolve_auth_mode

    try:
        stated = load_config().auth_mode
    except (FileNotFoundError, ValueError):
        stated = Config().auth_mode
    return resolve_auth_mode(stated)


@pytest.fixture(scope="session")
def drive(auth_mode):
    """A client pointed at the test document, the way the CLI points one.

    Skips rather than errors when the configured mode has no credential on this
    machine. These tests call Drive, and a missing credential is a machine that
    cannot run them, not a failure.
    """
    from gdoc.auth import DEFAULT_KEY_PATH, drive_service
    from gdoc.oauth import DEFAULT_TOKEN_PATH

    needed = DEFAULT_TOKEN_PATH if auth_mode == "oauth" else DEFAULT_KEY_PATH
    if not needed.exists():
        pytest.skip(f"no credential for auth_mode {auth_mode} at {needed}")
    return drive_service(doc_ids=TEST_DOC_ID)


@pytest.fixture(scope="session")
def test_doc_id():
    return TEST_DOC_ID


@pytest.fixture(scope="session")
def other_doc_id():
    return OTHER_DOC_ID


@pytest.fixture
def service_account_only(auth_mode):
    if auth_mode != "service_account":
        pytest.skip("asserts what Google refuses under Commenter")


@pytest.fixture
def oauth_only(auth_mode):
    if auth_mode != "oauth":
        pytest.skip("asserts what the guard refuses under OAuth")
