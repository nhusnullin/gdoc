from tools.gdoc.pairing import Pairing, add_version, find_by_doc_id, read_pairing, write_pairing

PAIRED = """\
---
title: Safeguarding policy
gdoc: 1AbC
gdoc_synced: 2026-08-13
gdoc_versions:
  - id: 1XyZ
    created: 2026-08-14
---

# Safeguarding policy

Body text.
"""

UNPAIRED = """\
---
title: Some note
---

Body.
"""

NO_FRONTMATTER = "# Just a heading\n\nBody.\n"

BODY_WITH_DASHES = """\
---
title: Fenced body
gdoc: 1AbC
---

# Heading

First paragraph.

---

Second paragraph.
"""


def test_reads_an_existing_pairing(tmp_path):
    path = tmp_path / "policy.md"
    path.write_text(PAIRED)
    pairing = read_pairing(path)
    assert pairing.doc_id == "1AbC"
    assert pairing.synced == "2026-08-13"
    assert pairing.versions == ({"id": "1XyZ", "created": "2026-08-14"},)


def test_returns_none_when_frontmatter_has_no_gdoc_key(tmp_path):
    path = tmp_path / "note.md"
    path.write_text(UNPAIRED)
    assert read_pairing(path) is None


def test_returns_none_when_there_is_no_frontmatter(tmp_path):
    path = tmp_path / "plain.md"
    path.write_text(NO_FRONTMATTER)
    assert read_pairing(path) is None


def test_write_preserves_other_frontmatter_keys_and_the_body(tmp_path):
    path = tmp_path / "policy.md"
    path.write_text(PAIRED)
    write_pairing(path, Pairing(doc_id="1AbC", synced="2026-08-20", versions=()))
    text = path.read_text()
    assert "title: Safeguarding policy" in text
    assert "2026-08-20" in text
    assert "Body text." in text


def test_write_adds_frontmatter_to_a_file_that_has_none(tmp_path):
    path = tmp_path / "plain.md"
    path.write_text(NO_FRONTMATTER)
    write_pairing(path, Pairing(doc_id="1New", synced="2026-08-20", versions=()))
    text = path.read_text()
    assert text.startswith("---\n")
    assert "gdoc: 1New" in text
    assert "Just a heading" in text


def test_add_version_returns_a_new_pairing_and_leaves_the_original_alone():
    original = Pairing(doc_id="1AbC", synced="2026-08-13", versions=())
    updated = add_version(original, doc_id="1XyZ", created="2026-08-14")
    assert original.versions == ()
    assert updated.versions == ({"id": "1XyZ", "created": "2026-08-14"},)
    assert updated is not original


def test_find_by_doc_id_locates_the_paired_file(tmp_path):
    (tmp_path / "sub").mkdir()
    target = tmp_path / "sub" / "policy.md"
    target.write_text(PAIRED)
    (tmp_path / "other.md").write_text(UNPAIRED)
    assert find_by_doc_id(tmp_path, "1AbC") == target


def test_find_by_doc_id_returns_none_when_nothing_matches(tmp_path):
    (tmp_path / "other.md").write_text(UNPAIRED)
    assert find_by_doc_id(tmp_path, "1AbC") is None


def test_write_preserves_body_containing_a_horizontal_rule(tmp_path):
    path = tmp_path / "doc.md"
    path.write_text(BODY_WITH_DASHES)
    write_pairing(path, Pairing(doc_id="1AbC", synced="2026-08-20", versions=()))
    text = path.read_text()
    assert "First paragraph." in text
    assert "Second paragraph." in text
    pairing = read_pairing(path)
    assert pairing.doc_id == "1AbC"


def test_write_clears_gdoc_synced_when_synced_is_none(tmp_path):
    path = tmp_path / "policy.md"
    path.write_text(PAIRED)  # has gdoc_synced: 2026-08-13
    write_pairing(path, Pairing(doc_id="1AbC", synced=None, versions=()))
    text = path.read_text()
    assert "gdoc_synced" not in text
    pairing = read_pairing(path)
    assert pairing.synced is None


def test_write_preserves_the_existing_file_mode(tmp_path):
    # The atomic write goes through mkstemp, which creates 0600. Without an
    # explicit chmod the rename would silently tighten permissions on a real note.
    md = tmp_path / "note.md"
    md.write_text("---\ngdoc: 1AbC\n---\n\nbody\n")
    md.chmod(0o644)
    write_pairing(md, Pairing(doc_id="1AbC", synced="2026-08-13"))
    assert md.stat().st_mode & 0o777 == 0o644
