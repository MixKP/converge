---
target: /booking
total_score: 30
p0_count: 0
p1_count: 1
p2_count: 3
p3_count: 1
timestamp: 2026-06-02T04-29-48Z
slug: src-views-bookingview-vue
---
# UX Critique: /booking

## Design Health Score

| # | Heuristic | Score | Key Issue |
|---|-----------|-------|-----------|
| 1 | Visibility of System Status | 3 | No loading state during Smart Suggestions generation |
| 2 | Match System / Real World | 4 | Natural language throughout |
| 3 | User Control and Freedom | 3 | Can close AI panel, but no way to exit mid-flow and return |
| 4 | Consistency and Standards | 4 | Cohesive with design system |
| 5 | Error Prevention | 3 | Smart suggestions require subject first, good; but time slot validation could be clearer |
| 6 | Recognition Rather than Recall | 3 | Selected slots visible, but calendar hides in AI mode |
| 7 | Flexibility and Efficiency | 2 | No keyboard shortcuts; only one path to book |
| 8 | Aesthetic and Minimalist Design | 3 | Clean, but technical watermark adds noise |
| 9 | Error Recovery | 3 | Clear error messages, inline validation |
| 10 | Help and Documentation | 2 | No contextual help; users must figure out flow |
| **Total** | | **30/40** | **Good** |

## Anti-Patterns Verdict

**LLM assessment**: Does NOT look AI-generated. The interface has clear intent, a specific Japanese Industrial color palette, and thoughtful progressive disclosure. The technical watermark is a deliberate stylistic choice that signals "engineering precision" rather than generic SaaS decoration.

**Deterministic scan**: The detector found 18 warnings, but critically:
- **Bounce easing warnings are NOT in BookingView.vue/BookingView.css** — they're in PageLayout.vue, style.css, SubmitButton.vue, and other views. BookingView correctly uses ease-out-quart.
- **Overused font warnings** — Instrument Sans is the committed display font per DESIGN.md, not a reflex choice.
- **Side-tab border warning** — in style.css for calendar events, not this view.

## Overall Impression

A solid, professional booking interface with two clear paths (manual selection vs. Smart Suggestions). The design system alignment is strong. The biggest opportunity is in the hidden context problem: when Smart Suggestions opens, users lose sight of the calendar they were just viewing.

## What's Working

1. **Progressive disclosure done right** — Subject → Branch → Teacher sequence guides users naturally.
2. **Smart Suggestions as an alternative path** — Doesn't replace manual flow; offers a faster route.
3. **Accessibility investment visible** — ARIA labels, focus-visible styles, reduced motion support.

## Priority Issues

### [P1] Calendar Context Lost in AI Mode
**Why it matters**: When users click "Smart Suggestions," the calendar disappears. They can't cross-reference availability against suggestions.

**Fix**: Keep the calendar visible at reduced width/opacity when AI panel opens.

**Suggested command**: `/impeccable layout BookingView.vue`

### [P2] No Loading State During Suggestion Generation
**Why it matters**: No visible feedback while API processes.

**Fix**: Add loading spinner or button state change during generation.

**Suggested command**: `/impeccable animate BookingView.vue`

### [P2] "Add 0 slot to cart" Button Confusion
**Why it matters**: Label reads "Add 0 slot to cart" at empty state — feels broken.

**Fix**: Change to "Select time slots to add" or hide until slots selected.

**Suggested command**: `/impeccable clarify BookingView.vue`

### [P2] No Progress Indicator
**Why it matters**: Users don't know how far along they are.

**Fix**: Add subtle progress indicator (dots or steps) near heading.

**Suggested command**: `/impeccable delight BookingView.vue`

### [P3] Technical Watermark Is Noise
**Why it matters**: Adds visual complexity without serving user need.

**Fix**: Remove or make much more subtle.

**Suggested command**: `/impeccable quieter BookingView.vue`

## Persona Red Flags

**Jordan (First-Timer)**: Two booking paths visible with no explanation of which to use.

**Sam (Accessibility-Dependent)**: Asterisk announced as separate element from label in screen reader.

**Riley (Stress Tester)**: No edge case handling for duplicate time slots.

## Minor Observations

- Time slot dropdowns have 27 options each — upper bound of working memory.
- "Smart Suggestions" could be more descriptive.
- Empty cart state not visible.
