import pytest

TEST_DOC_ID = "1tyVhOTw9-bJ99IJTBZoTfWAT6fm3rjGIzfpHQX91hkw"

# A real Drive id shape that is not the test document. Every request to it must
# be refused inside the process, so it is never fetched and never has to exist.
OTHER_DOC_ID = "1NotTheDocumentTheseTestsWereGiven00000000"


@pytest.fixture(scope="session")
def auth_mode():
    """The configured mode, or the default when there is no config."""
    from gdoc.config import Config, load_config

    try:
        return load_config().auth_mode
    except (FileNotFoundError, ValueError):
        return Config().auth_mode


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
