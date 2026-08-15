"""Finding pandoc, wherever it is.

The path was hardcoded to `/opt/homebrew/bin/pandoc`, which exists on one Mac and
nowhere else. `gdoc` has to run where the source documents live, so a path that
assumes Homebrew is a bug rather than a shortcut.
"""

from unittest.mock import patch

import pytest

from gdoc import export


def test_pandoc_is_found_on_the_path():
    with patch("shutil.which", return_value="/usr/local/bin/pandoc"):
        assert export.find_pandoc() == "/usr/local/bin/pandoc"


def test_the_homebrew_path_is_a_fallback_not_the_answer():
    """A machine with no pandoc on PATH but with the Homebrew install still works."""
    with patch("shutil.which", return_value=None), \
         patch("pathlib.Path.is_file", return_value=True):
        assert export.find_pandoc() == export.HOMEBREW_PANDOC


def test_the_path_wins_over_the_fallback():
    """Whatever the caller has installed beats a hardcoded guess."""
    with patch("shutil.which", return_value="/opt/local/bin/pandoc"), \
         patch("pathlib.Path.is_file", return_value=True):
        assert export.find_pandoc() == "/opt/local/bin/pandoc"


def test_a_missing_pandoc_is_reported_with_what_to_do():
    with patch("shutil.which", return_value=None), \
         patch("pathlib.Path.is_file", return_value=False):
        with pytest.raises(export.PandocNotFound) as caught:
            export.find_pandoc()
    message = str(caught.value)
    assert "pandoc" in message
    assert "PATH" in message


def test_the_module_no_longer_hardcodes_a_machine_specific_path():
    """PANDOC as a module constant was the bug. Nothing may import it."""
    assert not hasattr(export, "PANDOC"), (
        "PANDOC is back as a constant, so the path is fixed at import time again"
    )


def test_every_caller_resolves_at_call_time():
    """Resolving at import time defeats the fix on any machine that installs later."""
    import gdoc.generate
    import gdoc.render.body

    for module in (gdoc.generate, gdoc.render.body):
        assert not hasattr(module, "PANDOC"), (
            f"{module.__name__} still holds a module-level PANDOC"
        )
