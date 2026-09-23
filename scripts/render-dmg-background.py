#!/usr/bin/env python3
"""Render the macOS drag-install DMG background.

The Finder window content is 720×680 points. The PNG is 2× with a 144 DPI
tag so it stays sharp. Icon centers in scripts/package-dmg.sh must match
APP_ICON and APPLICATIONS_ICON.
"""

from pathlib import Path

from PIL import Image, ImageDraw, ImageFont

ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / "build" / "dmg-background.png"
APP_ICON = ROOT / "build" / "appicon.png"

WIDTH = 720
HEIGHT = 680
SCALE = 2

# Centers of the Finder icons, in window points. Keep in sync with package-dmg.sh.
APP_ICON_POS = (196, 430)
APPLICATIONS_ICON_POS = (524, 430)

CARD = (48, 252, 672, 632)  # left, top, right, bottom
CARD_RADIUS = 36


def px(n: float) -> int:
    return int(round(n * SCALE))


def lerp(a, b, t):
    return tuple(int(a[i] + (b[i] - a[i]) * t) for i in range(3))


def vertical_gradient(size, top, bottom):
    image = Image.new("RGB", size)
    draw = ImageDraw.Draw(image)
    height = size[1]
    for y in range(height):
        draw.line([(0, y), (size[0], y)], fill=lerp(top, bottom, y / (height - 1)))
    return image


def font(size, weight):
    face = ImageFont.truetype("/System/Library/Fonts/SFNS.ttf", px(size))
    # Width, optical size, GRAD, weight.
    face.set_variation_by_axes([100, 28, 400, weight])
    return face


def main():
    image = vertical_gradient((px(WIDTH), px(HEIGHT)), (26, 16, 72), (12, 9, 26))
    draw = ImageDraw.Draw(image)

    # A few specks in the upper left, matching the reference artwork.
    for x, y, r, color in (
        (78, 54, 2.2, (150, 140, 210)),
        (118, 92, 1.4, (120, 110, 180)),
        (64, 128, 1.2, (90, 80, 150)),
    ):
        draw.ellipse((px(x - r), px(y - r), px(x + r), px(y + r)), fill=color)

    left, top, right, bottom = CARD
    draw.rounded_rectangle(
        (px(left), px(top), px(right), px(bottom)),
        radius=px(CARD_RADIUS),
        fill=(246, 245, 251),
    )

    arrow_y = APP_ICON_POS[1]
    color = (92, 78, 196)
    shaft = 3.25
    draw.rounded_rectangle(
        (
            px(328),
            px(arrow_y - shaft / 2),
            px(386),
            px(arrow_y + shaft / 2),
        ),
        radius=px(shaft / 2),
        fill=color,
    )
    draw.polygon(
        (
            (px(378), px(arrow_y - 11)),
            (px(396), px(arrow_y)),
            (px(378), px(arrow_y + 11)),
        ),
        fill=color,
    )

    mark = Image.open(APP_ICON).convert("RGBA").resize((px(72), px(72)), Image.Resampling.LANCZOS)
    title = "akproxy"
    title_font = font(52, 560)
    subtitle = 'To install, drag into "Applications folder"'
    subtitle_font = font(16, 430)

    title_box = draw.textbbox((0, 0), title, font=title_font)
    title_w = title_box[2] - title_box[0]
    gap = px(16)
    row_w = mark.width + gap + title_w
    row_x = (image.width - row_w) // 2
    row_y = px(108)
    image.paste(mark, (row_x, row_y), mark)
    title_y = row_y + (mark.height - (title_box[3] - title_box[1])) // 2 - title_box[1]
    draw.text((row_x + mark.width + gap, title_y), title, font=title_font, fill=(255, 255, 255))

    sub_box = draw.textbbox((0, 0), subtitle, font=subtitle_font)
    sub_w = sub_box[2] - sub_box[0]
    sub_x = (image.width - sub_w) // 2
    sub_y = row_y + mark.height + px(18)
    draw.text((sub_x, sub_y), subtitle, font=subtitle_font, fill=(186, 180, 214))

    image.save(OUT, dpi=(72 * SCALE, 72 * SCALE))


if __name__ == "__main__":
    main()
