"""The contents list is written by us, not by a field and not by LibreOffice.

Every check here runs offline. No credential, no network, no LibreOffice. That is
the point of the module under test.
"""

import re
import zipfile
from xml.sax.saxutils import unescape

import pytest

from gdoc.render import build, contents
from gdoc.render.profiles import DEFAULT_TEMPLATE, TEMPLATES_DIR, resolve

EXAMPLE = TEMPLATES_DIR / "altery-group-policy-v1.0" / "example.md"

# Sections named in the master template's stale cached field result. They are
# also the master's own heading names, so they cannot tell a stale entry from one
# we wrote. They only guard the premise: the master still has a stale field.
GHOSTS = (
    "Policy Statements 1 - Details",
    "Variation of procedure for legal entity X",
    "Documentation and Record Keeping",
)

# A stale entry is found by its shape, not by its words. The master's placeholder
# section names are its real heading names too, so the same words legitimately
# appear in the entries we write. What cannot survive is the template's own entry:
# a hyperlink into Google's heading bookmarks, and the master's right tab at 12000
# twips. Each appears 19 times in the master and only inside its contents list.
STALE_ENTRY_MARKS = ('w:anchor="_heading=h.', 'w:pos="12000"')


@pytest.fixture(scope="module")
def built(tmp_path_factory):
    out = tmp_path_factory.mktemp("built") / "source.docx"
    build(EXAMPLE, out)
    return out


@pytest.fixture(scope="module")
def stale(tmp_path_factory):
    """A copy of the bundled master template, with its stale field intact.

    build() now writes the contents list itself, so a document it produces
    already carries real entries, not the master's stale ones. The tests that
    check contents.write's own behaviour on a stale field need the master
    template directly instead: it carries real Heading1-3 paragraphs, the
    stale cached field with the GHOSTS text, and no TOC1 style. Copied into
    tmp_path so a test never writes into the installed package.
    """
    out = tmp_path_factory.mktemp("stale") / "template.docx"
    master = resolve(DEFAULT_TEMPLATE)
    out.write_bytes(master.read_bytes())
    return out


def document_xml(path):
    return zipfile.ZipFile(path).read("word/document.xml").decode("utf8")


def styles_xml(path):
    return zipfile.ZipFile(path).read("word/styles.xml").decode("utf8")


def entry_paragraphs(path):
    xml = document_xml(path)
    return [p for p in re.findall(r"<w:p\b.*?</w:p>", xml, re.S)
            if re.search(r'w:pStyle w:val="TOC\d"', p)]


def field_span(path):
    """The document.xml the contents field spans, cached result included."""
    xml = document_xml(path)
    located = contents.field(xml)
    return xml[located.start:located.end]


def paragraph_texts(fragment):
    """The <w:t> text of each paragraph. Strict, so no field instruction leaks in."""
    return ["".join(re.findall(r"<w:t(?:\s[^>]*)?>([^<]*)</w:t>", paragraph))
            for paragraph in re.findall(r"<w:p\b.*?</w:p>", fragment, re.S)]


def test_headings_are_found_in_document_order(built):
    found = contents.headings(document_xml(built))
    assert [text for _level, text in found][:4] == [
        "1-Purpose", "1.1-Policy Owner", "1.1.1-Delegation", "2-Scope"
    ]
    assert [level for level, _text in found][:4] == [1, 2, 3, 1]


def test_the_source_build_still_carries_the_stale_field(stale):
    """Guards the premise. If this fails, the master changed and so must the fix."""
    xml = document_xml(stale)
    assert "<w:instrText" in xml
    assert any(ghost in xml for ghost in GHOSTS)


def test_writing_replaces_the_field_with_one_entry_per_heading(built, tmp_path):
    out = tmp_path / "written.docx"
    result = contents.write(built, out)
    assert len(result.entries) == len(entry_paragraphs(out))
    assert len(result.entries) == len(contents.headings(document_xml(built)))


def test_no_stale_template_section_survives(stale, tmp_path):
    """The template's own contents entries must not reach the output, anywhere.

    Runs against the master, which is the only fixture that still carries a stale
    cached result. build() writes the contents list itself now, so a document it
    produces has nothing stale left to leak, and this check would pass against a
    write that did nothing at all.
    """
    before = document_xml(stale)
    assert all(mark in before for mark in STALE_ENTRY_MARKS), "the master changed"
    out = tmp_path / "written.docx"
    contents.write(stale, out)
    after = document_xml(out)
    for mark in STALE_ENTRY_MARKS:
        assert mark not in after, f"a template contents entry survived: {mark}"


def test_the_field_is_kept_so_a_reader_can_still_refresh_it(built, tmp_path):
    """Replacing the cached result, not the field.

    Delete the field and Google imports plain text: no table of contents object,
    no update button, and a heading added later never appears. Keeping the field
    means the published document still holds a real, refreshable contents list.
    """
    out = tmp_path / "written.docx"
    contents.write(built, out)
    xml = document_xml(out)
    assert "<w:instrText" in xml
    assert 'w:fldCharType="begin"' in xml
    assert 'w:fldCharType="separate"' in xml
    assert 'w:fldCharType="end"' in xml


def test_the_field_brackets_the_entries(built, tmp_path):
    out = tmp_path / "written.docx"
    contents.write(built, out)
    xml = document_xml(out)
    begin = xml.index('w:fldCharType="begin"')
    end = xml.index('w:fldCharType="end"')
    first_entry = xml.index('w:pStyle w:val="TOC1"')
    assert begin < end
    assert begin < xml.rindex('w:pStyle w:val="TOC') < end
    assert first_entry < end


def test_every_entry_links_to_its_heading(built, tmp_path):
    out = tmp_path / "written.docx"
    result = contents.write(built, out)
    xml = document_xml(out)
    anchors = re.findall(r'<w:hyperlink w:anchor="([^"]+)"', xml)
    bookmarks = re.findall(r'<w:bookmarkStart[^>]*w:name="([^"]+)"', xml)
    ours = [a for a in anchors if a.startswith(contents.BOOKMARK_PREFIX)]
    assert len(ours) == len(result.entries)
    for anchor in ours:
        assert anchor in bookmarks, f"entry links to {anchor}, which no heading defines"


def test_each_heading_gets_exactly_one_bookmark(stale, tmp_path):
    out = tmp_path / "written.docx"
    result = contents.write(stale, out)
    xml = document_xml(out)
    starts = re.findall(rf'<w:bookmarkStart[^>]*w:name="({contents.BOOKMARK_PREFIX}\d+)"', xml)
    assert len(starts) == len(result.entries)
    assert len(set(starts)) == len(starts), "duplicate bookmark names"
    for name in starts:
        bookmark_id = re.search(
            rf'<w:bookmarkStart w:id="(\d+)" w:name="{name}"/>', xml).group(1)
        assert f'<w:bookmarkEnd w:id="{bookmark_id}"/>' in xml


def test_no_stale_cached_entry_survives_inside_the_field(stale, tmp_path):
    """The field stays, but its old cached result must not.

    Runs against the master for the same reason as the check above. The stale
    entries carry the master's own page numbers, which run past the end of the
    document, so the test is that every paragraph inside the field is an entry we
    wrote, naming a heading of this document and nothing else.
    """
    out = tmp_path / "written.docx"
    result = contents.write(stale, out)
    inside = field_span(out)
    for mark in STALE_ENTRY_MARKS:
        assert mark not in inside, f"a stale cached entry survived: {mark}"
    assert paragraph_texts(inside) == [
        contents.escape(text) for _level, text in result.entries
    ]


def test_toc_styles_are_added_when_the_template_lacks_them(stale, tmp_path):
    assert 'w:styleId="TOC1"' not in styles_xml(stale)
    out = tmp_path / "written.docx"
    contents.write(stale, out)
    after = styles_xml(out)
    for sid in ("Index", "TOC1", "TOC2", "TOC3"):
        assert f'w:styleId="{sid}"' in after


def test_entries_declare_no_font_and_no_bold(built, tmp_path):
    """Google applies Arial and bold level one when we leave this to its defaults.

    The entries must carry the TOC styles and nothing direct, so they inherit the
    document default, which is Calibri.
    """
    out = tmp_path / "written.docx"
    contents.write(built, out)
    for para in entry_paragraphs(out):
        assert "w:ascii=" not in para
        assert not re.search(r"<w:b(\s+w:val=\"(1|true|on)\")?\s*/>", para)


def test_entry_styles_declare_their_own_spacing(built, tmp_path):
    """Inheriting the template's spacing makes the page jump on the first refresh.

    Google imposes its own paragraph spacing when a reader refreshes the contents
    list. The template default computes to about 21.6pt per line; Google's refresh
    gives 16.4pt. Leave the styles to inherit and the contents list visibly tightens
    the first time anyone clicks update. Declaring the tighter value keeps it still.
    """
    out = tmp_path / "written.docx"
    contents.write(built, out)
    after = styles_xml(out)
    for sid in ("TOC1", "TOC2", "TOC3"):
        style = re.search(rf'<w:style [^>]*w:styleId="{sid}".*?</w:style>', after, re.S).group(0)
        assert "<w:spacing" in style, f"{sid} would inherit the template's looser spacing"


def test_entry_indents_match_what_a_refresh_produces(built, tmp_path):
    """Levels sit 18pt and 36pt in, because that is what Google's refresh uses.

    The house template used 283 and 567 twips. Close, but a refresh moves the
    entries sideways, which is exactly the visible jump this avoids.
    """
    out = tmp_path / "written.docx"
    contents.write(built, out)
    after = styles_xml(out)
    for sid, expected in (("TOC1", 0), ("TOC2", 360), ("TOC3", 720)):
        style = re.search(rf'<w:style [^>]*w:styleId="{sid}".*?</w:style>', after, re.S).group(0)
        assert f'w:left="{expected}"' in style, f"{sid} should indent {expected} twips"


def test_only_the_first_entry_carries_space_above(built, tmp_path):
    """Google leaves 3pt above the first entry and nothing above the rest.

    Word and Google add space-before to space-after rather than collapsing them, so
    putting this on the style would widen every gap instead of just the first.
    """
    out = tmp_path / "written.docx"
    contents.write(built, out)
    paras = entry_paragraphs(out)
    assert 'w:before="60"' in paras[0]
    for para in paras[1:]:
        assert 'w:before="60"' not in para


def test_page_numbers_are_written_when_supplied(built, tmp_path):
    out = tmp_path / "written.docx"
    contents.write(built, out, pages={"1-Purpose": 4, "2-Scope": 7})
    paras = entry_paragraphs(out)
    purpose = next(p for p in paras if "1-Purpose" in p)
    scope = next(p for p in paras if "2-Scope" in p)
    assert "<w:t>4</w:t>" in purpose
    assert "<w:t>7</w:t>" in scope


def test_a_heading_with_no_page_number_gets_an_empty_cell(built, tmp_path):
    """Better a blank than a wrong number. build() has no pagination to offer."""
    out = tmp_path / "written.docx"
    contents.write(built, out, pages={})
    paras = entry_paragraphs(out)
    assert paras
    assert all(re.search(r"<w:tab/><w:t[^>]*></w:t>", p) for p in paras)


def test_levels_map_to_their_styles(built, tmp_path):
    """Strict <w:t> matching, or the first entry is skipped.

    The first entry paragraph also carries the field instruction. A loose pattern
    matches <w:instrText> too, and then no entry text matches, so the check passes
    while verifying nothing.
    """
    out = tmp_path / "written.docx"
    result = contents.write(built, out)
    paras = entry_paragraphs(out)
    assert len(paras) == len(result.entries)
    checked = 0
    for (level, text), para in zip(result.entries, paras):
        found = "".join(re.findall(r"<w:t(?:\s[^>]*)?>([^<]*)</w:t>", para))
        assert found.startswith(text), f"expected {text!r} at the start of {found!r}"
        assert f'w:pStyle w:val="TOC{min(level, 3)}"' in para
        checked += 1
    assert checked == len(result.entries)


def test_a_document_with_no_headings_is_refused(built, tmp_path):
    stripped = tmp_path / "no_headings.docx"
    zin = zipfile.ZipFile(built)
    xml = document_xml(built).replace('w:pStyle w:val="Heading', 'w:pStyle w:val="Plain')
    with zipfile.ZipFile(stripped, "w") as zout:
        for item in zin.infolist():
            data = xml.encode("utf8") if item.filename == "word/document.xml" else zin.read(item.filename)
            zout.writestr(item, data)
    with pytest.raises(contents.ContentsError):
        contents.write(stripped, tmp_path / "out.docx")


def _with_document_xml(source, dst, xml):
    """A copy of source carrying the document.xml given."""
    with zipfile.ZipFile(source) as zin, zipfile.ZipFile(dst, "w") as zout:
        for item in zin.infolist():
            data = xml.encode("utf8") if item.filename == "word/document.xml" else zin.read(item.filename)
            zout.writestr(item, data)
    return dst


def _document_with(source, dst, old, new):
    """A copy of source whose document.xml has `old` replaced by `new`."""
    return _with_document_xml(source, dst, document_xml(source).replace(old, new))


def test_xml_special_characters_in_a_heading_survive(built, tmp_path):
    """A heading's own text is already escaped, so escaping it again is visible.

    Two effects, both of which the reader sees. The entry reads
    `Risk &amp; "Controls"` while the heading above it reads `Risk & "Controls"`.
    And the doubly escaped string is the key used to look up the page number,
    which the PDF carries with a literal ampersand, so the entry loses its page
    number too.
    """
    source = _document_with(
        built, tmp_path / "special.docx",
        '<w:t xml:space="preserve">Purpose</w:t>',
        '<w:t xml:space="preserve">Risk &amp; &quot;Controls&quot;</w:t>',
    )
    heading = next(text for _level, text in contents.headings(document_xml(source))
                   if "Risk" in text)
    assert heading == '1-Risk & "Controls"'

    out = tmp_path / "written.docx"
    contents.write(source, out, pages={heading: 5})
    entry = next(p for p in entry_paragraphs(out) if "Risk" in p)
    written = "".join(re.findall(r"<w:t(?:\s[^>]*)?>([^<]*)</w:t>", entry))
    assert unescape(written, {"&quot;": '"', "&apos;": "'"}) == f"{heading}5"
    assert "&amp;amp;" not in entry, "the heading text was escaped twice"


def test_rewriting_in_place_leaves_one_file(built, tmp_path):
    """A zip cannot be read and rewritten at the same path at once."""
    target = tmp_path / "inplace.docx"
    target.write_bytes(built.read_bytes())
    result = contents.rewrite_in_place(target, pages={"1-Purpose": 4})
    assert list(tmp_path.iterdir()) == [target], "a temporary file was left behind"
    assert len(result.entries) == 12
    assert "<w:t>4</w:t>" in document_xml(target)


def test_rewriting_in_place_cleans_up_after_a_failure(tmp_path):
    """The temporary file must not survive a failed rewrite."""
    junk = tmp_path / "junk.docx"
    junk.write_bytes(b"not a docx")
    with pytest.raises(zipfile.BadZipFile):
        contents.rewrite_in_place(junk)
    assert list(tmp_path.iterdir()) == [junk], "a temporary file was left behind"


def test_writing_twice_leaves_one_bookmark_per_heading_with_unique_names(built, tmp_path):
    """A rewrite must replace its own bookmarks, not add to them.

    The two-pass publish flow this feature exists for necessarily rewrites a
    document that already has a contents list: build writes blanks, the real
    page numbers come later. Bookmark names are unique document-wide, so
    writing twice must not leave two bookmarks carrying the same name.
    """
    once = tmp_path / "once.docx"
    twice = tmp_path / "twice.docx"
    result = contents.write(built, once)
    contents.write(once, twice)
    xml = document_xml(twice)
    starts = re.findall(rf'<w:bookmarkStart[^>]*w:name="({contents.BOOKMARK_PREFIX}\d+)"', xml)
    assert len(starts) == len(result.entries)
    assert len(set(starts)) == len(starts), "duplicate bookmark names"


def test_writing_twice_leaves_one_contents_entry_per_heading(built, tmp_path):
    """The second write replaces the first write's entries, not adds to them."""
    once = tmp_path / "once.docx"
    twice = tmp_path / "twice.docx"
    contents.write(built, once)
    result = contents.write(once, twice)
    assert len(result.entries) == len(entry_paragraphs(twice))


def test_a_bookmark_not_ours_survives_a_write_untouched(built, tmp_path):
    """Only our own bookmarks are ours to replace on a rewrite.

    The template's own bookmarks, or anyone else's, must be left alone. The
    attribute order here is Google's own, which is what the bundled master
    writes. Our order puts w:id first, so a regex anchored on that finds none of
    the template's bookmarks and every id we pick collides with one of them.
    """
    source = tmp_path / "with_foreign_bookmark.docx"
    xml = document_xml(built)
    foreign_start = (
        '<w:bookmarkStart w:colFirst="0" w:colLast="0" '
        'w:name="_heading=h.gjdgxs" w:id="8"/>'
    )
    foreign_end = '<w:bookmarkEnd w:id="8"/>'
    marker = xml.index("<w:body>") + len("<w:body>")
    xml = xml[:marker] + foreign_start + foreign_end + xml[marker:]
    with zipfile.ZipFile(built) as zin, zipfile.ZipFile(source, "w") as zout:
        for item in zin.infolist():
            data = xml.encode("utf8") if item.filename == "word/document.xml" else zin.read(item.filename)
            zout.writestr(item, data)

    out = tmp_path / "written.docx"
    contents.write(source, out)
    result_xml = document_xml(out)
    assert foreign_start in result_xml
    assert foreign_end in result_xml


def bookmark_ids(xml, tag):
    return re.findall(rf'<w:bookmark{tag}\b[^>]*?\bw:id="(\d+)"[^>]*/>', xml)


def test_writing_twice_keeps_every_bookmark_paired_and_every_id_unique(built, tmp_path):
    """Two invalid shapes that both still parse, so only counting finds them.

    Our ids must not collide with the template's Google-written ones. If they do,
    stripping our bookmarks on the second write also takes the foreign bookmark's
    end tag, and the template is left with a bookmarkStart that nothing closes.
    """
    once = tmp_path / "once.docx"
    twice = tmp_path / "twice.docx"
    contents.write(built, once)
    contents.write(once, twice)
    xml = document_xml(twice)
    starts = bookmark_ids(xml, "Start")
    ends = bookmark_ids(xml, "End")
    assert len(starts) == len(ends), "a bookmark was left unterminated"
    assert len(set(starts)) == len(starts), "duplicate bookmark ids"
    assert sorted(starts) == sorted(ends), "a start and an end disagree on their id"


# ------------------------------------------------- which field is the field ---
# Neither shape below is reachable with today's Google-exported master, which has
# exactly one instrText, one begin and one end. Both become reachable the day the
# master is re-saved from Word, and each produces a document whose body is
# immaculate and whose contents page lies, which is the defect this design exists
# to prevent. So they are held here rather than left to the day it happens.

FOREIGN_PAGE_FIELD = (
    "<w:p><w:r>"
    '<w:fldChar w:fldCharType="begin"/>'
    '<w:instrText xml:space="preserve"> PAGE </w:instrText>'
    '<w:fldChar w:fldCharType="separate"/></w:r>'
    "<w:r><w:t>7</w:t></w:r>"
    '<w:r><w:fldChar w:fldCharType="end"/></w:r></w:p>'
)

PAGEREF_ENTRY = (
    "<w:p>"
    '<w:r><w:t xml:space="preserve">1. Purpose</w:t></w:r>'
    '<w:r><w:fldChar w:fldCharType="begin"/>'
    '<w:instrText xml:space="preserve"> PAGEREF _heading=h.1fob9te \\h </w:instrText>'
    '<w:fldChar w:fldCharType="separate"/></w:r>'
    "<w:r><w:t>4</w:t></w:r>"
    '<w:r><w:fldChar w:fldCharType="end"/></w:r></w:p>'
)


def test_a_foreign_field_earlier_in_the_body_is_not_the_contents_field(stale, tmp_path):
    """Taking the first instruction splices the entries over a PAGE field.

    The real contents list is then left where it is, the output carries two
    contents lists, and write reports success. The field is chosen by its
    instruction naming TOC instead. This also holds the older rule that an end
    marker before the instruction, which this PAGE field has, is not the field's.
    """
    source = _document_with(stale, tmp_path / "page_field.docx",
                            "<w:body>", "<w:body>" + FOREIGN_PAGE_FIELD)
    out = tmp_path / "written.docx"
    result = contents.write(source, out)
    after = document_xml(out)
    for mark in STALE_ENTRY_MARKS:
        assert mark not in after, "the real contents list was left in place"
    assert " PAGE " in after, "the foreign field was destroyed"
    assert len(entry_paragraphs(out)) == len(result.entries)


def test_a_cached_entry_carrying_its_own_field_does_not_end_the_field(stale, tmp_path):
    """A refreshed TOC gives every entry a PAGEREF field of its own.

    That is what Word writes whenever a human clicks update. Taking the first end
    marker after the instruction then stops the field at the first entry: two
    paragraphs are replaced, the rest of the stale list survives, and the reader
    gets our entries followed by the template's. Counted by nesting depth instead.
    """
    xml = document_xml(stale)
    instruction = next(m for m in re.finditer(r"<w:p\b.*?</w:p>", xml, re.S)
                       if "<w:instrText" in m.group(0))
    source = _with_document_xml(
        stale, tmp_path / "pageref.docx",
        xml[: instruction.end()] + PAGEREF_ENTRY + xml[instruction.end():],
    )
    out = tmp_path / "written.docx"
    result = contents.write(source, out)
    after = document_xml(out)
    for mark in STALE_ENTRY_MARKS:
        assert mark not in after, "the stale list survived the write"
    # 19 paragraphs of stale cached result in the master, plus the one injected.
    assert result.replaced_paragraphs == 20, "the field stopped at the first entry"
    assert paragraph_texts(field_span(out)) == [
        contents.escape(text) for _level, text in result.entries
    ]


def test_a_field_with_no_toc_instruction_is_refused(stale, tmp_path):
    """A field is not the contents field just because it comes first."""
    source = _document_with(stale, tmp_path / "no_toc.docx", " TOC ", " PAGE ")
    with pytest.raises(contents.ContentsError, match="TOC"):
        contents.write(source, tmp_path / "written.docx")


def test_a_field_that_never_closes_is_refused(stale, tmp_path):
    """Unbalanced markers mean the end of the cached result is unknown."""
    source = _document_with(stale, tmp_path / "unclosed.docx",
                            'w:fldCharType="end"', 'w:fldCharType="begin"')
    with pytest.raises(contents.ContentsError, match="closed"):
        contents.write(source, tmp_path / "written.docx")
