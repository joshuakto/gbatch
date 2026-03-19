# Design System — gBatch

## Product Context
- **What this is:** A lightweight job scheduler CLI that wraps GCP services to replace UGER, giving teams a familiar `qsub`-like experience for submitting and monitoring compute workflows
- **Who it's for:** Researchers and engineers migrating from UGER, comfortable with CLI-first workflows
- **Space/industry:** HPC / bioinformatics / cloud compute infrastructure
- **Project type:** CLI tool with optional web dashboard
- **Reference products:** Seqera Platform (Nextflow Tower), dsub, Slurm, Google Cloud Batch

## Aesthetic Direction
- **Direction:** Industrial/Utilitarian
- **Decoration level:** Minimal — typography and spacing do the work
- **Mood:** Precise, trustworthy, efficient. Feels like modern infrastructure tooling (Railway, Vercel CLI) rather than legacy HPC
- **Reference sites:** seqera.io, railway.app, vercel.com/cli

## Typography
- **Display/Hero:** Geist 600-700 — clean, modern, designed for developer tools
- **Body:** Geist 400 — same family, excellent readability
- **UI/Labels:** Geist 500
- **Data/Tables:** Geist with tabular-nums — aligned numeric columns
- **Code/CLI:** Geist Mono — pairs perfectly, designed for terminals
- **Loading:** Google Fonts CDN (`https://fonts.googleapis.com/css2?family=Geist:wght@300;400;500;600;700&family=Geist+Mono:wght@400;500;600`)
- **Scale:**
  - xs: 12px / 0.75rem
  - sm: 14px / 0.875rem
  - base: 16px / 1rem
  - lg: 20px / 1.25rem
  - xl: 24px / 1.5rem
  - 2xl: 32px / 2rem
  - 3xl: 48px / 3rem

## Color
- **Approach:** Restrained (1 accent + neutrals, color is semantic)
- **Primary:** #2563EB (blue-600) — trust, infrastructure, links/actions
- **Accent:** #F59E0B (amber-500) — warnings, pending states
- **Neutrals:** Slate scale (cool grays)
  - 50: #F8FAFC
  - 100: #F1F5F9
  - 200: #E2E8F0
  - 300: #CBD5E1
  - 400: #94A3B8
  - 500: #64748B
  - 600: #475569
  - 700: #334155
  - 800: #1E293B
  - 900: #0F172A
- **Semantic:**
  - Success: #10B981 (completed jobs)
  - Warning: #F59E0B (pending, queue warnings)
  - Error: #EF4444 (failed jobs)
  - Info: #06B6D4 (system messages, autoscaling events)
- **Dark mode:** Invert surfaces (slate-900 bg, slate-800 cards), reduce saturation 10-15%, lighten primary to #3B82F6
- **CLI ANSI mapping:** green=success, red=error, yellow=pending, cyan=info, bold white=headers, dim=secondary text

## Spacing
- **Base unit:** 4px
- **Density:** Comfortable — HPC dashboards need breathing room between dense data
- **Scale:**
  - 2xs: 2px
  - xs: 4px
  - sm: 8px
  - md: 16px
  - lg: 24px
  - xl: 32px
  - 2xl: 48px
  - 3xl: 64px

## Layout
- **Approach:** Grid-disciplined — strict alignment for data tables, job lists, resource monitors
- **Grid:** 12-column at desktop (1200px+), 8-column tablet, 4-column mobile
- **Max content width:** 1200px
- **Border radius:**
  - sm: 4px (buttons, inputs, badges)
  - md: 8px (cards, panels)
  - lg: 12px (modals, terminal windows)
  - full: 9999px (status dots, avatars, pills)

## Motion
- **Approach:** Minimal-functional — only transitions that aid comprehension
- **Easing:** enter(ease-out) exit(ease-in) move(ease-in-out)
- **Duration:**
  - micro: 50-100ms (button hover, focus rings)
  - short: 150-250ms (status badge changes, tooltips)
  - medium: 250-400ms (panel transitions, loading states)
- **No entrance animations.** This is a tool people stare at while waiting for jobs.

## CLI Design Principles
- Job tables use fixed-width columns aligned with spaces (not tabs)
- Status words are colorized: DONE (green), FAILED (red), PENDING (yellow), RUNNING (cyan)
- Flags are blue, values are amber, prompts are green
- Error messages include actionable hints (e.g., "retry with --mem 128G")
- JSON output mode (`-o json`) for scripting/piping
- Summary lines at bottom of listings (e.g., "5 jobs | 3 done | 1 failed")
- **Status prefixes:** `✓` (green) for success, `✗` (red) for errors, `⚠` (amber) for warnings/degraded
- **Output structure:** Every command outputs: (1) status line, (2) primary data, (3) context/summary, (4) hint if applicable
- **Error template:** `✗ [What failed]: [human-readable reason]\n  Hint: [actionable command or next step]`
- **Warning template:** `⚠ [What's degraded]: [reason]. [what user still gets]`
- **Empty state template:** `[Friendly message]. [Primary action suggestion with example command]`
- **Loading:** Spinner + message for any operation >200ms. Format: `[action]...` (e.g., "Submitting to us-central1...")
- **TUI layout:** Split panel — job table (top), detail panel with log tail (bottom), status bar (top), key bar (bottom)
- **Terminal width:** Wide (≥120): full table. Medium (80-119): truncated columns. Narrow (<80): compact card format
- **TUI width:** ≥80 cols: split panel. <80 cols: table only, Enter for overlay detail view
- **Accessibility:** `NO_COLOR` env var disables all ANSI colors. `TERM=dumb` disables ANSI escapes entirely
- **Colorblind safety:** Status conveyed by text prefix + word, never color alone (✓ DONE, ✗ FAILED, ⚠ PENDING)
- **Unicode fallback:** If terminal doesn't support unicode, fall back: ✓→+, ✗→x, ⚠→!
- **Mount confirmation:** `gbatch submit --mount` shows "Mounts: gs://bucket → /path" in confirmation output
- **Interactive session (`gbatch ish`):**
  - Phase 1 (Create): spinner + VM spec + "⚠ VM will auto-terminate after 4 hours"
  - Phase 2 (Connect): "Connecting via SSH..." + mount summary if applicable
  - Phase 3 (Session): user's shell prompt. Show "Type 'exit' to end session and delete VM."
  - Phase 4 (Cleanup): "Deleting VM... ✓ VM deleted. Session lasted Xm. Est. cost: $X.XX"
  - Error: VM creation fails → error + hint. SSH fails → auto-delete VM + hint.
  - Orphan: `gbatch doctor` detects orphaned `gbatch-ish-*` VMs and prompts `Delete? [Y/n]`

## Decisions Log
| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-03-18 | Initial design system created | Created by /design-consultation based on competitive research (Seqera, dsub, Slurm) and industrial/utilitarian aesthetic direction |
| 2026-03-18 | Geist + Geist Mono over system fonts | Signals 'modern developer tool', distinguishes from legacy HPC UIs. ~40KB font load tradeoff accepted |
| 2026-03-18 | Amber for pending/warning over yellow | Warmer, more readable, elevates the palette beyond standard traffic-light semantics |
| 2026-03-18 | Minimal decoration in dashboard | Flat design with spacing and typography hierarchy instead of card shadows/borders everywhere |
| 2026-03-18 | Revised to 750 LOC minimal CLI | Reduced from 5000 LOC / GCP SDK to 750 LOC / shell out to gcloud. Ship small, grow from demand. |
| 2026-03-18 | gbatch ish 4-phase UX | Create → Connect → Session → Cleanup lifecycle with cost shown at exit and 4h auto-terminate warning |
| 2026-03-18 | Mount confirmation in submit output | Show "Mounts: gs://bucket → /path" so users can verify data access before job runs |
