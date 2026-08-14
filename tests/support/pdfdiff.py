#!/usr/bin/env python3
"""pdfdiff.py - automated visual diff between two PDFs.

Renders both PDFs to PNG (for eyeballing) and to raw PPM (for measuring),
then reports ImageMagick-equivalent metrics computed from the raw pixels:

  AE    absolute error   = number of pixels that differ at all (IM `-metric AE`)
  AE%   AE / total pixels
  RMSE  root mean squared error, normalised to 0..1 (IM `-metric RMSE`)
  MAXD  largest single-channel absolute difference (0..255)

Deliberately stdlib-only: pdftoppm writes binary P6 PPM, which is a 15-line
parser, so the harness needs neither Pillow nor ImageMagick.

Usage:
  pdfdiff.py A.pdf B.pdf --out DIR --label NAME [--pages 1,2] [--band top:0:0.12]
"""
import argparse
import os
import struct
import subprocess
import sys
import zlib

DPI = 150


def run(cmd):
    r = subprocess.run(cmd, capture_output=True)
    if r.returncode != 0:
        raise RuntimeError(f"{cmd!r} failed: {r.stderr.decode()[:400]}")
    return r.stdout


def n_pages(pdf):
    out = run(["pdfinfo", pdf]).decode()
    for line in out.splitlines():
        if line.startswith("Pages:"):
            return int(line.split()[1])
    raise RuntimeError("no page count")


def render(pdf, page, prefix, fmt):
    """Render one page. fmt is 'png' or 'ppm'. Returns the file path."""
    args = ["pdftoppm", "-r", str(DPI), "-f", str(page), "-l", str(page)]
    if fmt == "png":
        args.append("-png")
    args += [pdf, prefix]
    run(args)
    # pdftoppm zero-pads the page suffix to the width of the total page count
    for width in (1, 2, 3):
        p = f"{prefix}-{page:0{width}d}.{fmt}"
        if os.path.exists(p):
            return p
    raise RuntimeError(f"render missing for {pdf} p{page}")


def read_ppm(path):
    """Minimal binary P6 PPM reader -> (w, h, bytes)."""
    with open(path, "rb") as f:
        data = f.read()
    if not data.startswith(b"P6"):
        raise RuntimeError("not a P6 ppm")
    pos, fields = 2, []
    while len(fields) < 3:
        while pos < len(data) and data[pos : pos + 1].isspace():
            pos += 1
        if data[pos : pos + 1] == b"#":
            while data[pos : pos + 1] not in (b"\n", b""):
                pos += 1
            continue
        start = pos
        while pos < len(data) and not data[pos : pos + 1].isspace():
            pos += 1
        fields.append(int(data[start:pos]))
    pos += 1  # single whitespace after maxval
    w, h, _maxval = fields
    return w, h, data[pos : pos + w * h * 3]


def write_png(path, w, h, rgb):
    """Minimal PNG writer (stdlib zlib), used for the diff heat maps."""
    raw = b"".join(b"\x00" + rgb[y * w * 3 : (y + 1) * w * 3] for y in range(h))

    def chunk(tag, payload):
        return (
            struct.pack(">I", len(payload))
            + tag
            + payload
            + struct.pack(">I", zlib.crc32(tag + payload) & 0xFFFFFFFF)
        )

    with open(path, "wb") as f:
        f.write(b"\x89PNG\r\n\x1a\n")
        f.write(chunk(b"IHDR", struct.pack(">IIBBBBB", w, h, 8, 2, 0, 0, 0)))
        f.write(chunk(b"IDAT", zlib.compress(raw, 6)))
        f.write(chunk(b"IEND", b""))


def compare(pa, pb, diff_png=None, band=None):
    wa, ha, a = read_ppm(pa)
    wb, hb, b = read_ppm(pb)
    if (wa, ha) != (wb, hb):
        return {"error": f"size mismatch {wa}x{ha} vs {wb}x{hb}"}

    y0, y1 = 0, ha
    if band:
        y0, y1 = int(ha * band[0]), int(ha * band[1])

    ae = 0
    sq = 0
    maxd = 0
    npx = 0
    out = bytearray(wa * (y1 - y0) * 3) if diff_png else None

    for y in range(y0, y1):
        row = y * wa * 3
        for i in range(row, row + wa * 3, 3):
            d0 = a[i] - b[i]
            d1 = a[i + 1] - b[i + 1]
            d2 = a[i + 2] - b[i + 2]
            npx += 1
            if d0 or d1 or d2:
                ae += 1
                m = max(abs(d0), abs(d1), abs(d2))
                if m > maxd:
                    maxd = m
                sq += d0 * d0 + d1 * d1 + d2 * d2
                if out is not None:
                    j = (y - y0) * wa * 3 + (i - row)
                    out[j] = 255  # red = changed
            elif out is not None:
                j = (y - y0) * wa * 3 + (i - row)
                v = 255 - (255 - a[i]) // 5  # faint ghost of the original
                out[j] = out[j + 1] = out[j + 2] = v

    if diff_png and out is not None:
        write_png(diff_png, wa, y1 - y0, bytes(out))

    rmse = (sq / (npx * 3)) ** 0.5 / 255.0 if npx else 0.0
    return {
        "w": wa, "h": ha, "px": npx,
        "AE": ae, "AE_pct": 100.0 * ae / npx if npx else 0.0,
        "RMSE": rmse, "MAXD": maxd,
    }


BANDS = {
    "full":   None,
    "header": (0.00, 0.10),   # top 10% - logo / running head + rule
    "footer": (0.90, 1.00),   # bottom 10% - "For internal use only" + page no
}


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("a")
    ap.add_argument("b")
    ap.add_argument("--out", required=True)
    ap.add_argument("--label", required=True)
    ap.add_argument("--pages", default="1")
    ap.add_argument("--bands", default="full,header,footer")
    args = ap.parse_args()

    png_dir = os.path.join(args.out, "png")
    diff_dir = os.path.join(args.out, "diff")
    tmp_dir = os.path.join(args.out, ".ppm")
    for d in (png_dir, diff_dir, tmp_dir):
        os.makedirs(d, exist_ok=True)

    pages = [int(x) for x in args.pages.split(",")]
    na, nb = n_pages(args.a), n_pages(args.b)
    print(f"# {args.label}: A={os.path.basename(args.a)} ({na}p)  "
          f"B={os.path.basename(args.b)} ({nb}p)")
    print(f"{'page':>4} {'band':<8} {'AE':>10} {'AE%':>8} {'RMSE':>9} {'MAXD':>5}")

    rows = []
    for p in pages:
        if p > na or p > nb:
            print(f"{p:>4} {'--':<8} page missing (A={na}, B={nb})")
            continue
        pa_png = render(args.a, p, os.path.join(png_dir, f"{args.label}-A"), "png")
        pb_png = render(args.b, p, os.path.join(png_dir, f"{args.label}-B"), "png")
        pa = render(args.a, p, os.path.join(tmp_dir, f"{args.label}-A"), "ppm")
        pb = render(args.b, p, os.path.join(tmp_dir, f"{args.label}-B"), "ppm")
        for bname in args.bands.split(","):
            band = BANDS[bname]
            dp = os.path.join(diff_dir, f"{args.label}-p{p}-{bname}.png")
            r = compare(pa, pb, diff_png=dp, band=band)
            if "error" in r:
                print(f"{p:>4} {bname:<8} {r['error']}")
                continue
            print(f"{p:>4} {bname:<8} {r['AE']:>10} {r['AE_pct']:>7.3f}% "
                  f"{r['RMSE']:>9.5f} {r['MAXD']:>5}")
            rows.append((p, bname, r))
        _ = (pa_png, pb_png)
    return rows


if __name__ == "__main__":
    main()
