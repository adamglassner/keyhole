"""Draws Keyhole's icon and installer art as SVG and renders every size it needs.

Run: python3 tools/make_art.py   (needs rsvg-convert and Pillow)
"""
import math, os, subprocess, tempfile
from PIL import Image

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.join(HERE, "..")


def dial(cx, cy, r):
    ticks = []
    for i in range(60):
        a = i * math.pi / 30
        r1 = r * (0.85 if i % 5 == 0 else 0.9)
        w = r * (0.022 if i % 5 == 0 else 0.011)
        ticks.append(f'<line x1="{cx+math.cos(a)*r1:.1f}" y1="{cy+math.sin(a)*r1:.1f}" x2="{cx+math.cos(a)*r*0.97:.1f}" y2="{cy+math.sin(a)*r*0.97:.1f}" stroke="#5a3d10" stroke-width="{w:.1f}"/>')
    rivets = []
    for i in range(12):
        a = i * math.pi / 6 + math.pi / 12
        x, y = cx + math.cos(a) * r * 0.68, cy + math.sin(a) * r * 0.68
        rivets.append(f'<circle cx="{x:.1f}" cy="{y:.1f}" r="{r*0.045:.1f}" fill="#c9922e" stroke="#5a3d10" stroke-width="{r*0.012:.1f}"/><circle cx="{x-r*0.014:.1f}" cy="{y-r*0.014:.1f}" r="{r*0.015:.1f}" fill="#fbe7b0"/>')
    s = r * 0.22
    ky = cy - r * 0.05
    keyhole = (f'<circle cx="{cx}" cy="{ky-s*0.45:.1f}" r="{s*0.68:.1f}" fill="#120a02"/>'
               f'<path d="M{cx-s*0.36:.1f} {ky:.1f} L{cx+s*0.36:.1f} {ky:.1f} L{cx+s*0.62:.1f} {ky+s*1.45:.1f} L{cx-s*0.62:.1f} {ky+s*1.45:.1f}Z" fill="#120a02"/>'
               f'<circle cx="{cx}" cy="{ky-s*0.45:.1f}" r="{s*0.48:.1f}" fill="#ffd98a"/>'
               f'<path d="M{cx-s*0.22:.1f} {ky-s*0.1:.1f} L{cx+s*0.22:.1f} {ky-s*0.1:.1f} L{cx+s*0.4:.1f} {ky+s*1.25:.1f} L{cx-s*0.4:.1f} {ky+s*1.25:.1f}Z" fill="#ffd98a"/>'
               f'<circle cx="{cx}" cy="{ky-s*0.55:.1f}" r="{s*0.2:.1f}" fill="#fff6dc" opacity=".85"/>')
    return f'''
  <circle cx="{cx}" cy="{cy+r*0.05:.1f}" r="{r*1.02:.1f}" fill="rgba(0,0,0,.45)"/>
  <circle cx="{cx}" cy="{cy}" r="{r}" fill="url(#ring)"/>
  {''.join(ticks)}
  <circle cx="{cx}" cy="{cy}" r="{r*0.82:.1f}" fill="#3a2a0e"/>
  <circle cx="{cx}" cy="{cy}" r="{r*0.79:.1f}" fill="url(#plate)"/>
  {''.join(rivets)}
  <circle cx="{cx}" cy="{cy}" r="{r*0.47:.1f}" fill="none" stroke="#c9922e" stroke-width="{r*0.03:.1f}" opacity=".75"/>
  <circle cx="{cx}" cy="{cy}" r="{r*0.6:.1f}" fill="url(#glow)"/>
  {keyhole}'''


DEFS = '''<defs>
  <linearGradient id="ring" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#f6dc8e"/><stop offset=".45" stop-color="#c9922e"/><stop offset="1" stop-color="#6e4a14"/></linearGradient>
  <radialGradient id="plate" cx=".42" cy=".38" r=".75"><stop offset="0" stop-color="#3b4a4c"/><stop offset="1" stop-color="#162124"/></radialGradient>
  <radialGradient id="glow" cx=".5" cy=".5" r=".5"><stop offset="0" stop-color="#ffe6a3" stop-opacity=".7"/><stop offset="1" stop-color="#f4b942" stop-opacity="0"/></radialGradient>
  <linearGradient id="bg" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#16262a"/><stop offset="1" stop-color="#0a1214"/></linearGradient>
</defs>'''


def icon_svg(rounded=True):
    bg = '<rect width="1024" height="1024" rx="210" fill="url(#bg)"/>' if rounded else '<rect width="1024" height="1024" fill="url(#bg)"/>'
    return f'<svg xmlns="http://www.w3.org/2000/svg" width="1024" height="1024" viewBox="0 0 1024 1024">{DEFS}{bg}{dial(512, 512, 400)}</svg>'


def render(svg, w, h, out):
    with tempfile.NamedTemporaryFile("w", suffix=".svg", delete=False) as f:
        f.write(svg)
        p = f.name
    subprocess.run(["rsvg-convert", "-w", str(w), "-h", str(h), p, "-o", out], check=True)
    os.unlink(p)


def main():
    br = os.path.join(ROOT, "branding")
    gb = os.path.join(ROOT, "game", "assets", "branding")
    os.makedirs(br, exist_ok=True)
    os.makedirs(gb, exist_ok=True)
    big = os.path.join(br, "icon_1024.png")
    render(icon_svg(), 1024, 1024, big)
    im = Image.open(big).convert("RGBA")
    for s in (16, 32, 64, 256):
        im.resize((s, s), Image.LANCZOS).save(os.path.join(gb, f"icon_{s}.png"))
        im.resize((s, s), Image.LANCZOS).save(os.path.join(br, f"icon_{s}.png"))
    im.save(os.path.join(br, "keyhole.ico"), sizes=[(16, 16), (24, 24), (32, 32), (48, 48), (64, 64), (128, 128), (256, 256)])
    # Android launcher icons (legacy square + round)
    res = os.path.join(ROOT, "android", "app", "src", "main", "res")
    for name, s in (("mdpi", 48), ("hdpi", 72), ("xhdpi", 96), ("xxhdpi", 144), ("xxxhdpi", 192)):
        d = os.path.join(res, f"mipmap-{name}")
        os.makedirs(d, exist_ok=True)
        im.resize((s, s), Image.LANCZOS).save(os.path.join(d, "ic_launcher.png"))
        # round version: crop the dial into a circle
        sq = Image.new("RGBA", (1024, 1024))
        render(icon_svg(rounded=False), 1024, 1024, os.path.join(tempfile.gettempdir(), "kh_sq.png"))
        sq = Image.open(os.path.join(tempfile.gettempdir(), "kh_sq.png")).convert("RGBA")
        mask = Image.new("L", (1024, 1024), 0)
        from PIL import ImageDraw
        ImageDraw.Draw(mask).ellipse((0, 0, 1023, 1023), fill=255)
        sq.putalpha(mask)
        sq.resize((s, s), Image.LANCZOS).save(os.path.join(d, "ic_launcher_round.png"))
    # installer art
    header = f'''<svg xmlns="http://www.w3.org/2000/svg" width="300" height="114" viewBox="0 0 300 114">{DEFS}
      <rect width="300" height="114" fill="url(#bg)"/>{dial(57, 57, 44)}
      <text x="116" y="66" font-family="Poppins, DejaVu Sans, sans-serif" font-weight="700" font-size="34" letter-spacing="4" fill="#f6dc8e">KEYHOLE</text></svg>'''
    welcome = f'''<svg xmlns="http://www.w3.org/2000/svg" width="328" height="628" viewBox="0 0 328 628">{DEFS}
      <rect width="328" height="628" fill="url(#bg)"/>{dial(164, 250, 120)}
      <text x="164" y="440" text-anchor="middle" font-family="Poppins, DejaVu Sans, sans-serif" font-weight="700" font-size="44" letter-spacing="6" fill="#f6dc8e">KEYHOLE</text>
      <text x="164" y="476" text-anchor="middle" font-family="Poppins, DejaVu Sans, sans-serif" font-weight="600" font-size="14" letter-spacing="5" fill="#e9dcc0">THE DAILY WORD VAULT</text></svg>'''
    for svg, w, h, name in ((header, 150, 57, "installer_header.bmp"), (welcome, 164, 314, "installer_welcome.bmp")):
        tmp = os.path.join(tempfile.gettempdir(), "kh_art.png")
        render(svg, w * 2, h * 2, tmp)
        Image.open(tmp).convert("RGB").resize((w, h), Image.LANCZOS).save(os.path.join(br, name))
    print("art done")


if __name__ == "__main__":
    main()
