---
name: Converge
description: Professional booking system for tutoring sessions
colors:
  primary: "#3e4c7a"
  primary-deep: "#2d3954"
  accent-sage: "#7a8b6d"
  accent-sage-light: "#a3b094"
  accent-coral: "#c96d5d"
  bg-cream: "#faf9f6"
  bg-card: "#ffffff"
  bg-subtle: "#f0efe9"
  text-primary: "#1a1c23"
  text-secondary: "#585a63"
  text-muted: "#8a8c94"
  border-subtle: "#e8e6e0"
  border-medium: "#d0cdc5"
  border-strong: "#b0aca3"
typography:
  display:
    fontFamily: "'Instrument Sans', system-ui, sans-serif"
    fontSize: "clamp(2rem, 5vw, 3rem)"
    fontWeight: 600
    lineHeight: 1.2
  body:
    fontFamily: "'IBM Plex Sans', system-ui, sans-serif"
    fontSize: "1rem"
    fontWeight: 400
    lineHeight: 1.6
  label:
    fontFamily: "'JetBrains Mono', monospace"
    fontSize: "0.75rem"
    fontWeight: 500
    letterSpacing: "0.05em"
    textTransform: "uppercase"
rounded:
  sm: "4px"
  md: "6px"
  lg: "8px"
spacing:
  xs: "4px"
  sm: "8px"
  md: "16px"
  lg: "24px"
  xl: "32px"
components:
  button-primary:
    backgroundColor: "{colors.primary}"
    textColor: "#ffffff"
    rounded: "{rounded.md}"
    padding: "14px 24px"
  button-primary-hover:
    backgroundColor: "{colors.primary-deep}"
  input:
    backgroundColor: "{colors.bg-card}"
    textColor: "{colors.text-primary}"
    rounded: "{rounded.sm}"
    padding: "10px 14px"
---

# Design System: Converge

## 1. Overview

**Creative North Star: "The Clean Schedule"**

A professional booking system built on clarity, efficiency, and quiet confidence. The visual language speaks through restraint rather than decoration—every element earns its place through function, not ornament. The Japanese Industrial color palette brings warmth without sacrificing professionalism, while the typography hierarchy (Instrument Sans for display, IBM Plex Sans for body, JetBrains Mono for technical labels) creates clear information architecture at a glance.

This system rejects the dated, bureaucratic aesthetic of traditional scheduling software. No cluttered forms, no confusing navigation, no visual noise. The interface fades into the background so the user can focus on what matters: finding the right teacher at the right time.

**Key Characteristics:**
- **Quiet confidence:** Professional but approachable, formal but warm
- **Information-first:** Typography and spacing guide the eye before decoration
- **Restraint:** One primary accent (indigo) used deliberately, not sprayed everywhere
- **Motion with purpose:** Spring animations on interactions, not gratuitous transitions

## 2. Colors

A Japanese Industrial-inspired palette that balances warmth with professionalism. The cream background (#faf9f6) provides a paper-like foundation without crossing into the saturated warm-tint monoculture.

### Primary
- **Deep Muted Indigo-Navy** (#3e4c7a): The main action color—buttons, links, active states, and calendar events. Its desaturated nature keeps it professional while still being clearly interactive. Hover state deepens to #2d3954.

### Secondary
- **Sage Green** (#7a8b6d): Secondary accent for success states, availability indicators, and softer interactive elements. The light variant (#a3b094) serves for backgrounds and subtle highlights.

### Tertiary
- **Soft Coral** (#c96d5d): Used sparingly for attention states—warnings, important notifications, and the current-time indicator. Its warmth provides clear visual separation from the cooler primary/secondary palette.

### Neutral
- **Cream White** (#faf9f6): Main background. Warm enough to feel human, light enough to stay neutral.
- **Pure White** (#ffffff): Card backgrounds and the active calendar surface.
- **Subtle Warm Gray** (#f0efe9): Section dividers and elevated surface backgrounds.
- **Text Primary** (#1a1c23): Body and headings—deep enough for ≥7:1 contrast against cream.
- **Text Secondary** (#585a63): Supporting text, metadata, timestamps.
- **Text Muted** (#8a8c94): Placeholder text and disabled states (used sparingly).
- **Border Subtle** (#e8e6e0): Default borders and dividers.
- **Border Medium** (#d0cdc5): Interactive borders (inputs, buttons).
- **Border Strong** (#b0aca3): Focus indicators and emphasized borders.

### Named Rules
**The One Accent Rule.** The primary indigo appears on ≤15% of any screen. Its rarity creates visual hierarchy—when something is indigo, it's meant to be clicked or noted.

**The Warmth-Without-Heat Rule.** The cream background provides warmth, but the overall temperature remains cool-to-neutral through the indigo/sage dominance. No saturated yellows, oranges, or reds except coral for alerts.

## 3. Typography

**Display Font:** Instrument Sans (with system-ui fallback)
**Body Font:** IBM Plex Sans (with system-ui fallback)
**Label/Mono Font:** JetBrains Mono

**Character:** A confident pairing that feels engineered but not cold. Instrument Sans brings contemporary energy to headings; IBM Plex Sans provides warmth and readability at body sizes; JetBrains Mono signals technical precision for time labels and metadata.

### Hierarchy
- **Display** (600, clamp(2rem, 5vw, 3rem), 1.2): Page headings and main section titles.
- **Headline** (600, 1.5rem, 1.3): Section headings and card titles.
- **Title** (500, 1.125rem, 1.4): Subsection headings and emphasized labels.
- **Body** (400, 1rem, 1.6): All prose, form labels, and primary content. Max line length: 65–75ch.
- **Label** (500, 0.75rem, 0.05em letter-spacing, uppercase): Calendar day headers, time labels, metadata, and technical indicators.

### Named Rules
**The Mono Label Rule.** Time, dates, and metadata always use JetBrains Mono with uppercase tracking. This creates a visual shorthand for "this is structural information, not content."

## 4. Elevation

The system uses shadows as a response to state, not as decoration. Surfaces are flat at rest; shadows appear on hover, focus, and elevation to indicate interactivity and hierarchy.

### Shadow Vocabulary
- **Card Elevation** (`0 1px 3px rgba(58, 62, 74, 0.04), 0 4px 12px rgba(58, 62, 74, 0.03)`): Default card and container shadow—barely there, just enough to separate from background.
- **Elevated Surface** (`0 4px 16px rgba(58, 62, 74, 0.06), 0 12px 32px rgba(58, 62, 74, 0.04)`): Modals, dropdowns, and hovered elements. Clear lift but still diffuse—no harsh drop shadows.
- **Focus Ring** (`0 0 0 3px rgba(62, 76, 122, 0.3)`): Keyboard navigation indicator—soft indigo glow, no double-border artifacts.

### Named Rules
**The Flat-By-Default Rule.** Surfaces rest flat. Shadows are a response to interaction (hover, focus, modal state), not a default styling choice.

## 5. Components

### Buttons
- **Shape:** Rounded corners (6px radius) — approachable but not playful.
- **Primary:** Indigo background (#3e4c7a), white text, 14px vertical padding with 24px horizontal. Spring animation on press, -2px lift on hover.
- **Hover / Focus:** Background deepens to #2d3954, elevation increases, subtle shine effect sweeps across on hover.
- **Disabled:** 40% opacity, no hover state, pointer-events: none.
- **Focus Ring:** 3px soft indigo glow (rgba(62, 76, 122, 0.3)) for keyboard navigation.

### Calendar Events
- **Style:** Gradient background (indigo, semi-transparent), left border accent (3px solid indigo), 8px padding.
- **State:** Hover lifts (-2px) and brightens gradient; active state adds sage green dot indicator.
- **Today Highlight:** Subtle indigo tint (8% opacity) on the current day cell—no jarring full-color background.

### Inputs / Fields
- **Style:** 1px border (starts medium gray, transitions to indigo on focus), 4px rounded corners, white background.
- **Focus:** Border shifts to indigo (#3e4c7a), 3px focus ring appears, subtle elevation.
- **Error / Disabled:** Error state uses coral accent; disabled uses 40% opacity and removes all hover effects.

### Cards / Containers
- **Corner Style:** 4–8px rounded corners depending on container type.
- **Background:** White for active surfaces, cream for nested containers, subtle warm gray for section dividers.
- **Shadow Strategy:** Card Elevation by default; upgrades to Elevated Surface on hover/focus.
- **Border:** No default borders; separation achieved through shadow and background contrast alone.
- **Internal Padding:** 16px (lg) for most containers; 24px (xl) for featured cards.

### Navigation
- **Style:** Minimal tab-based navigation—no visible borders, indigo underline on active state.
- **Typography:** IBM Plex Sans 500 weight, 0.875rem size.
- **States:** Default is text-secondary color; active is text-primary with indigo underline; hover shifts to primary color before active.
- **Mobile Treatment:** Bottom navigation bar on mobile, horizontal tabs on desktop.

## 6. Do's and Don'ts

### Do:
- **Do** use ≥7:1 contrast for all body text against its background. This is WCAG AAA territory—accessibility is not optional.
- **Do** use JetBrains Mono for all time labels, calendar headers, and metadata. It's the visual shorthand for structural information.
- **Do** add spring animations to interactive elements (buttons, cards, inputs). The 0.34, 1.56, 0.64, 1 cubic-bezier curve is the standard.
- **Do** use the cream background (#faf9f6) for the main page surface and white (#ffffff) for cards/containers. This separation creates subtle hierarchy.
- **Do** keep motion subtle by default—scale transforms max out at 1.02, translations at -2px. No bouncing, no elastic extremes.

### Don't:
- **Don't** use border-left or border-right greater than 1px as a colored accent stripe. This is the side-stripe border anti-pattern—use background tints or full borders instead.
- **Don't** use gradient text. It's decorative, never meaningful. Use solid colors with weight/size for emphasis.
- **Don't** add identical card grids repeating endlessly. Vary the hierarchy—featured cards, list items, and compact views should coexist.
- **Don't** add tiny uppercase tracked eyebrows above every section. One deliberate eyebrow is voice; one on every section is AI grammar.
- **Don't** let text overflow its container. Test all clamp() values at narrow viewports; reduce the max size or rewrite copy if it overflows.
- **Don't** make the interface look dated or bureaucratic. No dense grids, no small gray text on light gray backgrounds, no cluttered forms.
- **Don't** use the saturated warm-tint monoculture (cream/sand/paper/parchment at OKLCH L 0.84-0.97, C < 0.06, hue 40-100). Our cream #faf9f6 is warm but clearly a neutral, not a color statement.
