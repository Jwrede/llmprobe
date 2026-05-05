"""Create a polished hero image from the TUI screenshot."""
from PIL import Image, ImageDraw, ImageFont

def rounded_rect_mask(size, radius):
    mask = Image.new("L", size, 0)
    d = ImageDraw.Draw(mask)
    d.rounded_rectangle([(0, 0), (size[0] - 1, size[1] - 1)], radius=radius, fill=255)
    return mask

def make_thumbnail(screenshot_path, output_path):
    tui = Image.open(screenshot_path).convert("RGBA")
    tui = tui.resize((1400, 890), Image.LANCZOS)

    W, H = 1600, 1000
    bg = Image.new("RGBA", (W, H), (10, 14, 20, 255))

    # Subtle radial glow
    glow = Image.new("RGBA", (W, H), (0, 0, 0, 0))
    gd = ImageDraw.Draw(glow)
    cx, cy = W // 2, H // 2
    for r in range(600, 0, -2):
        alpha = int(25 * (r / 600))
        gd.ellipse([cx - r, cy - r, cx + r, cy + r], fill=(60, 140, 255, alpha))
    bg = Image.alpha_composite(bg, glow)

    # Round corners on TUI screenshot
    mask = rounded_rect_mask(tui.size, 16)
    tui_rounded = Image.new("RGBA", tui.size, (0, 0, 0, 0))
    tui_rounded.paste(tui, mask=mask)

    # Place screenshot centered, shifted down slightly
    sx = (W - tui.size[0]) // 2
    sy = (H - tui.size[1]) // 2 + 30

    # Dim the screenshot so text pops
    dimmed = tui_rounded.copy()
    dim_overlay = Image.new("RGBA", tui.size, (10, 14, 20, 160))
    dimmed = Image.alpha_composite(dimmed, dim_overlay)

    bg.paste(dimmed, (sx, sy), dimmed)

    # Draw a subtle border around the screenshot
    bd = ImageDraw.Draw(bg)
    bd.rounded_rectangle(
        [(sx - 1, sy - 1), (sx + tui.size[0], sy + tui.size[1])],
        radius=16, outline=(80, 130, 220, 100), width=2
    )

    draw = ImageDraw.Draw(bg)

    try:
        title_font = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSansMono-Bold.ttf", 72)
        sub_font = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf", 22)
        small_font = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf", 16)
    except OSError:
        title_font = ImageFont.load_default()
        sub_font = title_font
        small_font = title_font

    # Title
    title = "llmprobe"
    bbox = draw.textbbox((0, 0), title, font=title_font)
    tw = bbox[2] - bbox[0]
    tx = (W - tw) // 2
    ty = H // 2 - 70

    # Shadow
    for dx, dy in [(2, 2), (1, 1), (3, 3)]:
        draw.text((tx + dx, ty + dy), title, font=title_font, fill=(0, 0, 0, 180))
    draw.text((tx, ty), title, font=title_font, fill=(255, 255, 255, 255))

    # Tagline
    tagline = "Probe LLM API endpoints. Measure TTFT, latency, throughput."
    bbox2 = draw.textbbox((0, 0), tagline, font=sub_font)
    tw2 = bbox2[2] - bbox2[0]
    draw.text(((W - tw2) // 2, ty + 90), tagline, font=sub_font, fill=(170, 180, 200, 230))

    # Stats
    stats = "6 models  |  60,000 probes  |  7 days  |  zero SDKs"
    bbox3 = draw.textbbox((0, 0), stats, font=small_font)
    tw3 = bbox3[2] - bbox3[0]
    draw.text(((W - tw3) // 2, ty + 128), stats, font=small_font, fill=(110, 130, 170, 200))

    final = bg.convert("RGB")
    final.save(output_path, quality=95, optimize=True)
    print(f"Saved {output_path} ({W}x{H})")


if __name__ == "__main__":
    make_thumbnail("demo/tui-screenshot.png", "demo/tui-thumbnail.png")
