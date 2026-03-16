# lsbot Logo Design Spec

**Date:** 2026-03-16
**Status:** Approved

---

## Summary

A new logo and visual identity for **lsbot** (Lean Secure Bot), reflecting both its Unix heritage (`ls` command) and its security-first, local-first philosophy.

---

## Brand Concept

- **ls** = Unix's `ls` command — the first command you run on a new machine; lean, foundational, always present
- **l** = Lean
- **s** = Secure
- **bot** = the product

The `ls` mark inside a shield encodes the dual meaning visually: Unix roots + security protection.

---

## Logo Elements

### Symbol
- **Shape:** Sharp pentagon shield (classic heraldic proportions, pointed bottom, straight sides)
- **Fill:** Solid accent color at low opacity (~7%) as background wash
- **Stroke:** Accent color, 1.8px, `stroke-linejoin: round`
- **Content:** `ls` in monospace, centred inside the shield

### Wordmark
- **Text:** `bot`
- **Font:** `'Courier New', monospace`
- **Weight:** 700 (bold)
- **Color (dark bg):** `#e6edf3` (near-white)
- **Color (light bg):** `#0f172a` (near-black)
- **Letter-spacing:** -1px

### Full lockup
Shield icon to the left, `bot` wordmark to the right, vertically centred on the shield midpoint. Slogans sit below the wordmark, left-aligned to it.

---

## Color Palette

| Token | Dark bg | Light bg | Usage |
|---|---|---|---|
| Accent (primary) | `#4ade80` | `#16a34a` | Shield stroke, `ls` text, primary slogan |
| Shield fill | `rgba(74,222,128,0.07)` | `rgba(74,222,128,0.12)` | Shield background wash |
| Wordmark | `#e6edf3` | `#0f172a` | `bot` text |
| Primary slogan | `#4ade80` | `#16a34a` | "Lean & Secure Bot" |
| Secondary slogan | `#374151` | `#9ca3af` | "Your AI, Your Data." |
| Background (primary) | `#0d1117` | `#ffffff` | Canvas |

**Rationale:** Sage green (`#4ade80`) — softer than pure terminal green, readable at small sizes, distinctive without the clichéd "hacker" connotation of `#00ff41`.

---

## Typography

All type uses **monospace** exclusively (`'Courier New', monospace`):
- Monospace is the deliberate choice — it signals Unix, terminal, and precision
- No mixing with sans-serif (rejected in design review — monospace throughout is more cohesive)

| Element | Size | Weight | Letter-spacing |
|---|---|---|---|
| `ls` inside shield | 22px | 700 | +1px |
| `bot` wordmark | 42px | 700 | -1px |
| Primary slogan | 11.5px | 700 | +1.5px |
| Secondary slogan | 9.5px | 400 | +1.2px |

---

## Slogans

| Priority | Text | Treatment |
|---|---|---|
| Primary | **Lean & Secure Bot** | Accent color, bold, 11.5px — prominent |
| Secondary | Your AI, Your Data. | Dimmed grey, regular weight, 9.5px — subdued |

The primary slogan states what it is. The secondary states the value proposition. The hierarchy must be preserved in all uses.

---

## Variants

### 1. Full logo (horizontal lockup)
Shield + `bot` + two-line slogan. Use in: README header, website hero, splash screens.

### 2. Compact logo (no slogan)
Shield + `bot` only. Use in: navigation bars, sub-headers, inline references.

### 3. Icon only
Shield with `ls` only. Use in: favicon (16/32/64px), app icon, GitHub avatar.

### 4. README banner
Icon + stacked wordmark/slogan, left-bordered with accent color (`border-left: 3px solid #4ade80`).

---

## Files to Produce

| File | Format | Variant |
|---|---|---|
| `assets/logo/lsbot-logo-dark.svg` | SVG | Full logo, dark |
| `assets/logo/lsbot-logo-light.svg` | SVG | Full logo, light |
| `assets/logo/lsbot-icon.svg` | SVG | Icon only |
| `assets/logo/lsbot-banner-dark.svg` | SVG | README banner, dark |
| `assets/logo/lsbot-banner-light.svg` | SVG | README banner, light |
| `assets/logo/lsbot-icon-64.png` | PNG | Favicon 64px raster |
| `assets/logo/lsbot-icon-32.png` | PNG | Favicon 32px raster |
| `assets/logo/lsbot-icon-16.png` | PNG | Favicon 16px raster |

---

## Usage Rules

1. Never place the dark-background logo on a light background without switching to the light variant
2. Never change the font — monospace is non-negotiable
3. Never recolour `ls` or the shield to anything other than the defined accent tokens
4. Minimum icon size: 16px (shield + `ls` remain legible)
5. Clear space: minimum 16px on all sides of the full lockup; minimum 8px on all sides of the icon-only variant
