import pytest

TEST_DOC_ID = "1tyVhOTw9-bJ99IJTBZoTfWAT6fm3rjGIzfpHQX91hkw"


@pytest.fixture(scope="session")
def drive():
    from tools.gdoc.auth import drive_service

    return drive_service()


@pytest.fixture(scope="session")
def test_doc_id():
    return TEST_DOC_ID
