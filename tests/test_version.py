import pytest

from gdoc.version import __version__, satisfies_minimum


def test_version_is_a_three_part_number():
    parts = __version__.split(".")
    assert len(parts) == 3
    assert all(part.isdigit() for part in parts)


def test_satisfies_minimum_true_when_equal():
    assert satisfies_minimum("0.2.0", "0.2.0") is True


def test_satisfies_minimum_true_when_newer():
    assert satisfies_minimum("0.3.0", "0.2.0") is True
    assert satisfies_minimum("1.0.0", "0.9.9") is True


def test_satisfies_minimum_false_when_older():
    assert satisfies_minimum("0.1.0", "0.2.0") is False


def test_satisfies_minimum_compares_numerically_not_lexically():
    """"0.10.0" must beat "0.9.0"; string comparison would get this backwards."""
    assert satisfies_minimum("0.10.0", "0.9.0") is True


def test_satisfies_minimum_rejects_a_non_semver_string():
    with pytest.raises(ValueError):
        satisfies_minimum("not-a-version", "0.2.0")
