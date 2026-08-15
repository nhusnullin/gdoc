"""Confine every request to the files the command was given.

Under a service account the credential reached only the documents that had been
shared with it, and on those it was a Commenter. Two limits, both Google's: a
small reachable set, and no writing inside it. OAuth removes both, because the
token is Nail and the scope is full Drive. There is no narrower scope that reads
comments and writes replies, so the reachable set has to be narrowed here.

The rule is about files, never about methods. A verb allowlist would have left
every document Nail owns readable, which is the risk OAuth actually introduces,
and it would need editing every time the tool learns a new call.

The check sits in the HTTP transport rather than at the call sites, so it sees
every request the client makes: calls added later, and any other Google API
built on the same transport, which is how the Docs API is covered without the
call sites knowing it exists.
"""

import json
import re
from urllib.parse import unquote, urlsplit

CARRY = "carry"
LEARN = "learn"
REFUSE = "refuse"

_DRIVE_HOST = "www.googleapis.com"
_DOCS_HOST = "docs.googleapis.com"

# The three path shapes that name a file.
_FILE_PATHS = (
    re.compile(r"^/drive/v3/files/([^/]+)"),
    re.compile(r"^/upload/drive/v3/files/([^/]+)"),
    re.compile(r"^/v1/documents/([^/:]+)"),
)

# Creating a file names no id, because it is making one.
_CREATE_PATHS = frozenset({"/drive/v3/files", "/upload/drive/v3/files"})

# Neither does asking what the API looks like, or who is signed in.
_NO_FILE_PATHS = (
    re.compile(r"^/discovery/"),
    re.compile(r"^/drive/v3/about$"),
)

# Changing who else can reach a file is refused on every file, allowed or not.
# Granting other people access is a different authority from changing a
# document, and gdoc has no use for it.
#
# Reading the list is not refused here. It is a read of the named file like any
# other, and under a service account Google answers it with a 403 anyway, which
# tests/test_access_integration.py records.
_FORBIDDEN_PATH = re.compile(r"/permissions(/|$)")
_READ_METHODS = frozenset({"GET", "HEAD"})

# googleapiclient turns a GET whose URI is over MAX_URI_LENGTH into a POST
# carrying this header. The server acts on the override, so the guard must too,
# or an over-long files.list arrives as POST /drive/v3/files and reads as a
# create.
_OVERRIDE_HEADER = "x-http-method-override"


def _has_dot_segment(path: str) -> bool:
    """True when the path contains a . or .. segment, encoded or not.

    Google normalises these and answers 302 to the normalised path, and
    httplib2 follows that redirect inside the transport the guard wraps. So the
    file the guard reads out of the path is not the file the server acts on.
    Verified live on 2026-08-15: both /files/{a}/../{b} and its %2e%2e form
    redirect to /files/{b}.

    They are refused rather than resolved. gdoc never builds such a path, so
    there is nothing to lose by refusing, and resolving would mean trusting
    that our normalisation matches Google's exactly.
    """
    return any(segment in (".", "..") for segment in unquote(path).split("/"))


def file_id(uri: str) -> str | None:
    """The file a request addresses, or None when it names none.

    The Docs API writes documents/{id}:batchUpdate, so the id stops at the
    colon.
    """
    path = urlsplit(uri).path
    for pattern in _FILE_PATHS:
        found = pattern.match(path)
        if found:
            return found.group(1)
    return None


def effective_method(method: str, headers=None) -> str:
    """The method the server will act on.

    An override header wins, because that is what googleapiclient sets when it
    rewrites a long GET as a POST, and what the server honours.
    """
    for name, value in (headers or {}).items():
        if name.lower() == _OVERRIDE_HEADER and value:
            return str(value).upper()
    return (method or "GET").upper()


def verdict(method: str, uri: str, allowed) -> str:
    """CARRY, LEARN or REFUSE.

    LEARN is a create: carry it, then take the id out of the response. Three
    outcomes rather than two, because generate addresses files that did not
    exist when the client was built.

    The query string is ignored, so fields and uploadType cannot change a
    verdict. The host is matched, so a familiar path shape elsewhere does not
    pass.
    """
    method = (method or "GET").upper()
    split = urlsplit(uri)
    if split.hostname not in (_DRIVE_HOST, _DOCS_HOST):
        return REFUSE
    path = split.path
    if _has_dot_segment(path):
        return REFUSE
    if _FORBIDDEN_PATH.search(path) and method not in _READ_METHODS:
        return REFUSE
    named = file_id(uri)
    if named is not None:
        return CARRY if named in allowed else REFUSE
    if path in _CREATE_PATHS and method == "POST":
        return LEARN
    if any(pattern.match(path) for pattern in _NO_FILE_PATHS):
        return CARRY
    # files.list, batch, and anything else that names no file.
    return REFUSE


class GuardedHttp:
    """An httplib2-shaped transport that reaches only the files it was given.

    The signature mirrors google_auth_httplib2.AuthorizedHttp.request, verified
    against the installed 0.4.1. Everything other than request is proxied,
    because googleapiclient reaches for more than request during media upload.

    The allowed set grows through exactly two doors: the ids handed in at
    construction, and the ids that creates this object carried came back with.
    Never add a third.
    """

    def __init__(self, inner, allowed=frozenset()):
        self._inner = inner
        self.allowed = frozenset(allowed)

    def request(
        self,
        uri,
        method="GET",
        body=None,
        headers=None,
        redirections=5,
        connection_type=None,
        **kwargs,
    ):
        decision = verdict(effective_method(method, headers), uri, self.allowed)
        if decision == REFUSE:
            self._refuse(method, uri)
        response, content = self._inner.request(
            uri,
            method=method,
            body=body,
            headers=headers,
            redirections=redirections,
            connection_type=connection_type,
            **kwargs,
        )
        if decision == LEARN:
            self._learn(response, content)
        return response, content

    def _refuse(self, method, uri):
        path = urlsplit(uri).path
        named = file_id(uri)
        if named:
            why = f"{named} is not a file it was given"
        else:
            why = "that request names no single file"
        raise PermissionError(
            f"gdoc refused {(method or 'GET').upper()} {path}. "
            f"It reaches only the files it was given, and {why}."
        )

    def _learn(self, response, content):
        """Add the id a create came back with.

        Never raises, and the except is broad for that reason. A body that is
        not JSON, a response object shaped unlike httplib2's, or a create that
        failed all teach nothing, and the next call on that file is refused.
        That is the safe direction: a stray temporary file beats a widened set.
        """
        try:
            status = getattr(response, "status", None)
            if status is None and hasattr(response, "get"):
                status = response.get("status")
            if not str(status or "").startswith("2"):
                return
            created = json.loads(content)
            new_id = created.get("id") if isinstance(created, dict) else None
        except Exception:  # noqa: BLE001 - see the docstring
            return
        if new_id:
            self.allowed = self.allowed | {new_id}

    def __getattr__(self, name):
        """Proxy everything else to the wrapped transport.

        _inner is fetched through the instance dict rather than self._inner,
        because on a half-built instance, one being copied for example, the
        attribute is absent and self._inner would call this method again.
        """
        try:
            inner = object.__getattribute__(self, "_inner")
        except AttributeError:
            raise AttributeError(name) from None
        return getattr(inner, name)
