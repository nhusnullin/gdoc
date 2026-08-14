"""Ask Google which page each heading landed on.

A contents list needs page numbers, and page numbers do not exist until something
lays the document out. Headless LibreOffice used to be that something. Google Docs
is also that something, and we are uploading to it anyway.

So `gdoc generate` uploads once with blank page numbers, exports the result as PDF,
reads which page each heading actually sits on, writes those numbers into the
contents list, and uploads the version it publishes. Google is the layout engine,
which means the page numbers describe the document the reader is holding rather
than a local approximation of it.

Writing the numbers can in principle move the text that follows, so the second
pass is checked against its own export with `drift`. In measured runs the numbers
were stable on the first try, because a page number is one or two characters at the
right margin.

`resolve` is kept free of subprocesses so it can be tested against page text
directly. `from_pdf` is the thin shell around poppler.
"""

import re
import subprocess
from pathlib import Path

PDFTOTEXT = "pdftotext"
PDFINFO = "pdfinfo"
CONTENTS_TITLE = "Contents"
# A heading long enough to wrap gets matched on this many leading characters.
PREFIX = 40


class PaginationError(RuntimeError):
    """The rendered document could not be read, so page numbers are unavailable."""


def normalise(text: str) -> str:
    return re.sub(r"\s+", " ", text).strip()


def resolve(page_lines, headings) -> dict:
    """Map heading text to the 1-based page its body heading sits on.

    page_lines is one list of text lines per page, in order.

    Matched line by line, never as a substring of the page. A heading such as
    "Appendices" also occurs inside ordinary prose, and a substring match puts it
    on whichever page mentions it first. A heading absent from every page is left
    out of the result rather than guessed at.
    """
    pages = [[normalise(line) for line in lines if normalise(line)] for lines in page_lines]
    contents = next(
        (i for i, lines in enumerate(pages) if any(line == CONTENTS_TITLE for line in lines)),
        -1,
    )
    found = {}
    for _level, text in headings:
        target = normalise(text)
        page = _first_page_with_line(pages, contents + 1, target)
        if page is not None:
            found[text] = page
    return found


def _first_page_with_line(pages, start, target):
    for i in range(start, len(pages)):
        if any(line == target for line in pages[i]):
            return i + 1
    # A heading that wrapped across lines will not match whole. Allow its opening.
    head = target[:PREFIX]
    for i in range(start, len(pages)):
        if any(line.startswith(head) for line in pages[i]):
            return i + 1
    return None


def page_count(pdf: Path) -> int:
    result = subprocess.run([PDFINFO, str(pdf)], capture_output=True, text=True)
    if result.returncode != 0:
        raise PaginationError(f"{PDFINFO} failed on {pdf}:\n{result.stderr}")
    for line in result.stdout.splitlines():
        if line.startswith("Pages:"):
            return int(line.split()[1])
    raise PaginationError(f"{PDFINFO} reported no page count for {pdf}")


def page_lines(pdf: Path) -> list:
    """The text of each page, as a list of lines. Needs poppler's pdftotext."""
    pdf = Path(pdf)
    out = []
    for page in range(1, page_count(pdf) + 1):
        result = subprocess.run(
            [PDFTOTEXT, "-f", str(page), "-l", str(page), str(pdf), "-"],
            capture_output=True, text=True,
        )
        if result.returncode != 0:
            raise PaginationError(f"{PDFTOTEXT} failed on {pdf} page {page}:\n{result.stderr}")
        out.append(result.stdout.splitlines())
    return out


def from_pdf(pdf: Path, headings) -> dict:
    return resolve(page_lines(pdf), headings)


def drift(written: dict, published: dict) -> dict:
    """Headings whose published page differs from the number we wrote.

    Empty means the contents list describes the document it sits in. Anything else
    means another pass is needed, and saying so is the whole point.
    """
    keys = set(written) | set(published)
    return {
        k: (written.get(k), published.get(k))
        for k in keys
        if written.get(k) != published.get(k)
    }
