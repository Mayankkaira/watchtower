---
name: watchtower-frontend
description: "Frontend design system for Watchtower — an uptime monitoring SaaS. USE WHEN: building or modifying any HTML, CSS, or JS in the frontend/ directory. Enforces clean, professional design with zero emojis, proper semantic structure, SVG/CSS-only icons, and a dark-mode design system. Prevents AI-generated 'slop' patterns."
---

# Watchtower Frontend Skill

## Brand

- **Name**: Watchtower
- **Tagline**: Infrastructure monitoring that sees everything.
- **Tone**: Professional, confident, minimal. Not playful, not corporate-stuffy.
- **Logo mark**: SVG tower/shield icon — never Unicode characters or emojis.

## Design Principles

1. **No emojis. Ever.** Use inline SVGs or CSS-drawn indicators (colored dots, borders, background tints).
2. **Complete layouts.** Every section must have proper spacing, alignment, and visual completeness. No orphan elements, no half-finished grids.
3. **Restrained color.** Dark background (#0a0a0f), cards (#111117), subtle borders (#1a1a2e). Accent color sparingly — only for interactive elements and key data.
4. **Typography hierarchy.** Use font-weight and size to create hierarchy. Headlines: 600-700 weight, never 800. Body: 400. Labels: 500 + uppercase + letter-spacing.
5. **Whitespace is design.** Generous padding (2rem+ on sections), consistent gaps, breathing room between elements.
6. **Status uses colored dots only.** Green (#10b981), yellow (#f59e0b), red (#ef4444). Small circles (8-10px). No glow effects, no pulsing animations.
7. **Cards are flat.** 1px border, subtle background difference. No box-shadows, no hover transforms, no "glowing" effects.
8. **Data-first.** Dashboard shows numbers prominently. Clean tables/lists with aligned columns.
9. **Intentional motion.** Only transition opacity and color. No translateY hover effects on cards. Transitions ≤200ms.

## Color Tokens

```
--bg:          #0a0a0f
--surface:     #111117
--surface-2:   #1a1a24
--border:      #1f1f2e
--border-hover:#2a2a3d
--text:        #e4e4ec
--text-2:      #9898b0
--text-3:      #5a5a72
--accent:      #6366f1   (indigo)
--accent-hover:#818cf8
--green:       #10b981
--yellow:      #f59e0b
--red:         #ef4444
```

## Component Patterns

### Status Indicator
```html
<span class="status-dot status-dot--up"></span>
```
```css
.status-dot { width: 8px; height: 8px; border-radius: 50%; display: inline-block; }
.status-dot--up { background: var(--green); }
.status-dot--degraded { background: var(--yellow); }
.status-dot--down { background: var(--red); }
```

### Section Headers
```html
<div class="section-hd">
  <h2>Monitors</h2>
  <button class="btn btn--sm">Add Monitor</button>
</div>
```
No emojis, no badge pills that say "AI-Powered".

### Feature Cards (Landing)
Use a small SVG icon (20x20) or a short text label. Never emoji. Keep description to 2 lines max.

### Monitor List Items
Horizontal layout: status dot | name + url | response time | uptime % | mini uptime bars. Clean grid alignment.

### Pricing
3 columns, consistent height, left-aligned feature lists with simple check marks (SVG or CSS `::before` content).

## Anti-Patterns (NEVER DO)

- Emoji as icons (🧠🔗📉🔐📊🛠️ etc.)
- "Unique to X" tag badges on feature cards
- "AI-Powered" labels scattered around
- Glow/shadow effects (box-shadow with color spread)
- Animated pulsing on status indicators
- Social proof stats with inflated numbers
- Unicode symbols as logo (◆, ◉, etc.)
- Hover effects that move elements (translateY)
- Gradient text effects
- "How It Works" numbered step sections
- "Trusted by X teams" hero badges
