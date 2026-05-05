"""Convert tmux terminal capture (with ANSI codes) to a PNG screenshot."""
import re
import sys
from PIL import Image, ImageDraw, ImageFont

BG = (13, 17, 23)
DEFAULT_FG = (189, 194, 207)

SGR_COLORS = {
    30: (40, 42, 54),    31: (255, 85, 85),   32: (80, 250, 123),
    33: (241, 250, 140), 34: (98, 114, 255),  35: (255, 121, 198),
    36: (139, 233, 253), 37: (248, 248, 242),
    90: (98, 114, 164),  91: (255, 110, 110), 92: (105, 255, 148),
    93: (255, 255, 165), 94: (125, 139, 255), 95: (255, 146, 223),
    96: (164, 255, 255), 97: (255, 255, 255),
}

def xterm_256(n):
    if n < 16:
        std = [
            (0,0,0),(128,0,0),(0,128,0),(128,128,0),(0,0,128),(128,0,128),(0,128,128),(192,192,192),
            (128,128,128),(255,0,0),(0,255,0),(255,255,0),(0,0,255),(255,0,255),(0,255,255),(255,255,255),
        ]
        return std[n]
    if n < 232:
        n -= 16
        base = [0, 95, 135, 175, 215, 255]
        return (base[n // 36], base[(n // 6) % 6], base[n % 6])
    v = 8 + 10 * (n - 232)
    return (v, v, v)


def parse_sgr(params, fg):
    parts = params.split(";") if params else ["0"]
    i = 0
    while i < len(parts):
        try:
            code = int(parts[i])
        except ValueError:
            i += 1
            continue
        if code == 0:
            fg = DEFAULT_FG
        elif 30 <= code <= 37 or 90 <= code <= 97:
            fg = SGR_COLORS.get(code, DEFAULT_FG)
        elif code == 38:
            if i + 1 < len(parts) and parts[i + 1] == "5" and i + 2 < len(parts):
                fg = xterm_256(int(parts[i + 2]))
                i += 2
            elif i + 1 < len(parts) and parts[i + 1] == "2" and i + 4 < len(parts):
                fg = (int(parts[i + 2]), int(parts[i + 3]), int(parts[i + 4]))
                i += 4
        elif code == 39:
            fg = DEFAULT_FG
        i += 1
    return fg


def render(capture_path, output_path):
    with open(capture_path, "r") as f:
        raw_lines = f.readlines()

    mono = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf", 13)
    braille_font = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf", 13)

    char_w = mono.getbbox("M")[2]
    char_h = int(mono.getmetrics()[0] * 1.3)
    pad = 20

    cols = 160
    rows = len(raw_lines)

    img_w = cols * char_w + pad * 2
    img_h = rows * char_h + pad * 2
    img = Image.new("RGB", (img_w, img_h), BG)
    draw = ImageDraw.Draw(img)

    ansi_re = re.compile(r"\x1b\[([0-9;]*)m")

    for row, line in enumerate(raw_lines):
        line = line.rstrip("\n")
        fg = DEFAULT_FG
        col = 0
        pos = 0
        while pos < len(line):
            m = ansi_re.match(line, pos)
            if m:
                fg = parse_sgr(m.group(1), fg)
                pos = m.end()
                continue
            ch = line[pos]
            pos += 1
            if ch == "\x1b":
                while pos < len(line) and line[pos] not in "mHJK":
                    pos += 1
                if pos < len(line):
                    pos += 1
                continue

            if ch != " ":
                px = pad + col * char_w
                py = pad + row * char_h
                cp = ord(ch)
                if 0x2800 <= cp <= 0x28FF or 0x2500 <= cp <= 0x257F or 0x2580 <= cp <= 0x259F or 0x256D <= cp <= 0x2570:
                    draw.text((px, py), ch, font=braille_font, fill=fg)
                else:
                    draw.text((px, py), ch, font=mono, fill=fg)
            col += 1

    img.save(output_path, optimize=True)
    print(f"Saved {output_path} ({img_w}x{img_h})")


if __name__ == "__main__":
    capture = sys.argv[1] if len(sys.argv) > 1 else "demo/tui-capture.txt"
    output = sys.argv[2] if len(sys.argv) > 2 else "demo/tui-screenshot.png"
    render(capture, output)
