#!/usr/bin/env bash
# lo_toc.sh - update the Word TOC field (and PAGE fields) in a .docx using
# headless LibreOffice, re-saving in place as .docx.
#
#   ./lo_toc.sh /abs/path/to/file.docx            # update TOC, save .docx
#   ./lo_toc.sh /abs/path/to/file.docx out.pdf    # update TOC, export PDF too
#
# Uses a throwaway LibreOffice user profile so it never touches the user's
# own Basic macros or settings.
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Keep the throwaway profile out of the skill folder. The hub syncs over
# Nextcloud, and a 700KB LibreOffice profile would be pushed to every teammate.
PROFILE="${LO_PROFILE:-${TMPDIR:-/tmp}/altery-policy-loprofile}"
DOCX="$(cd "$(dirname "$1")" && pwd)/$(basename "$1")"
PDF="${2:-}"

mkdir -p "$PROFILE/user/basic/Standard"

# --- 1. bootstrap the profile if it is empty -------------------------------
if [ ! -f "$PROFILE/user/basic/script.xlc" ]; then
  soffice --headless --norestore \
    -env:UserInstallation="file://$PROFILE" --terminate_after_init >/dev/null 2>&1 || true
fi

# --- 2. install the macro module -------------------------------------------
BAS_SRC="$HERE/UpdateToc.bas"
python3 - "$BAS_SRC" "$PROFILE/user/basic/Standard/UpdateToc.xba" <<'PY'
import sys, xml.sax.saxutils as su
src, dst = sys.argv[1], sys.argv[2]
code = open(src, encoding='utf-8').read()
open(dst, 'w', encoding='utf-8').write(
 '<?xml version="1.0" encoding="UTF-8"?>\n'
 '<!DOCTYPE script:module PUBLIC "-//OpenOffice.org//DTD OfficeDocument 1.0//EN" "module.dtd">\n'
 '<script:module xmlns:script="http://openoffice.org/2000/script" '
 'script:name="UpdateToc" script:language="StarBasic">'
 + su.escape(code) + '</script:module>\n')
PY

cat > "$PROFILE/user/basic/Standard/script.xlb" <<'XLB'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE library:library PUBLIC "-//OpenOffice.org//DTD OfficeDocument 1.0//EN" "library.dtd">
<library:library xmlns:library="http://openoffice.org/2000/library" library:name="Standard" library:readonly="false" library:passwordprotected="false">
 <library:element library:name="Module1"/>
 <library:element library:name="UpdateToc"/>
</library:library>
XLB
[ -f "$PROFILE/user/basic/Standard/Module1.xba" ] || cat > "$PROFILE/user/basic/Standard/Module1.xba" <<'M1'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE script:module PUBLIC "-//OpenOffice.org//DTD OfficeDocument 1.0//EN" "module.dtd">
<script:module xmlns:script="http://openoffice.org/2000/script" script:name="Module1" script:language="StarBasic"></script:module>
M1

# --- 3. run it -------------------------------------------------------------
if [ -n "$PDF" ]; then
  PDF="$(cd "$(dirname "$PDF")" && pwd)/$(basename "$PDF")"
  DOCX_PATH="$DOCX" PDF_PATH="$PDF" soffice --headless --norestore --nolockcheck \
    -env:UserInstallation="file://$PROFILE" \
    "vnd.sun.star.script:Standard.UpdateToc.ToPdf?language=Basic&location=application"
else
  DOCX_PATH="$DOCX" soffice --headless --norestore --nolockcheck \
    -env:UserInstallation="file://$PROFILE" \
    "vnd.sun.star.script:Standard.UpdateToc.Run?language=Basic&location=application"
fi
