"""Create a polished hero image: TUI screenshot with rounded corners, blue border, text overlay."""
from PIL import Image, ImageDraw, ImageFont

def make_thumbnail(screenshot_path, output_path):
    tui = Image.open(screenshot_path).convert("RGBA")
    tui = tui.resize((1400, 890), Image.LANCZOS)

    pad = 3
    radius = 16
    W = tui.size[0] + pad * 2
    H = tui.size[1] + pad * 2

    canvas = Image.new("RGBA", (W, H), (0, 0, 0, 0))

    # Rounded mask for the screenshot
    mask = Image.new("L", tui.size, 0)
    md = ImageDraw.Draw(mask)
    md.rounded_rectangle([(0, 0), (tui.size[0] - 1, tui.size[1] - 1)], radius=radius, fill=255)

    tui_rounded = Image.new("RGBA", tui.size, (0, 0, 0, 0))
    tui_rounded.paste(tui, mask=mask)

    # Dim the screenshot for text contrast
    dim = Image.new("RGBA", tui.size, (10, 14, 20, 140))
    tui_dimmed = Image.alpha_composite(tui_rounded, dim)

    # Re-apply mask after dimming (edges stay clean)
    final_tui = Image.new("RGBA", tui.size, (0, 0, 0, 0))
    final_tui.paste(tui_dimmed, mask=mask)

    canvas.paste(final_tui, (pad, pad), final_tui)

    # Blue border with rounded corners
    bd = ImageDraw.Draw(canvas)
    bd.rounded_rectangle(
        [(1, 1), (W - 2, H - 2)],
        radius=radius + pad, outline=(80, 140, 240, 200), width=2
    )

    draw = ImageDraw.Draw(canvas)

    try:
        title_font = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSansMono-Bold.ttf", 72)
        sub_font = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf", 22)
        small_font = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf", 16)
    except OSError:
        title_font = ImageFont.load_default()
        sub_font = title_font
        small_font = title_font

    # Title centered
    title = "llmprobe"
    bbox = draw.textbbox((0, 0), title, font=title_font)
    tw = bbox[2] - bbox[0]
    tx = (W - tw) // 2
    ty = H // 2 - 70

    for dx, dy in [(2, 2), (1, 1), (3, 3)]:
        draw.text((tx + dx, ty + dy), title, font=title_font, fill=(0, 0, 0, 200))
    draw.text((tx, ty), title, font=title_font, fill=(255, 255, 255, 255))

    # Tagline
    tagline = "Probe LLM API endpoints. Measure TTFT, latency, throughput."
    bbox2 = draw.textbbox((0, 0), tagline, font=sub_font)
    tw2 = bbox2[2] - bbox2[0]
    draw.text(((W - tw2) // 2, ty + 90), tagline, font=sub_font, fill=(180, 190, 210, 240))

    canvas.save(output_path, optimize=True)
    print(f"Saved {output_path} ({W}x{H})")


if __name__ == "__main__":
    make_thumbnail("demo/tui-screenshot.png", "demo/tui-thumbnail.png")
