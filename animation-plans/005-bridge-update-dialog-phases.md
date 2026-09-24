# 005 — Bridge update dialog phase content

- **Status**: Implemented with Motion for React; native feel check pending. Original CSS steps superseded.
- **Commit**: f3ce0df
- **Severity**: MEDIUM
- **Category**: Missed opportunities; interruptibility
- **Estimated scope**: 2 files, about 20 lines
- **Depends on**: 001 for `--ease-out`

## Problem

`frontend/src/features/update-dialog.tsx:165-185` changes title and detail by phase. At `:201-276`, download progress, release notes, and action buttons are conditionally swapped. The shared `DialogPopup` already animates opening (`frontend/src/components/ui/dialog.tsx:89`), but an *open* dialog's phase content changes instantly. The `download-progress` event can fire frequently (`update-dialog.tsx:113-117`), so animating every progress value would be distracting.

```tsx
{phase === "downloading" ? (
  <div className="px-6 pb-2">
    <Progress value={progress}>
```

## Target

When a meaningful phase changes (`available`, `downloading`, `verifying`, `installing`, `ready`, `failed`), fade the text/content *in* once with `opacity: 0 → 1` over `150ms var(--ease-out)` (`cubic-bezier(0.23, 1, 0.32, 1)`). No position or scale motion, no exit wait, no fade on progress-number events, no animation on dialog reopening after dismissal when the phase is unchanged. For reduced motion use the same fade over `100ms`. Keep the dialog's existing centered 200ms open/close motion and the progress indicator's existing behavior.

## Repo conventions to follow

`frontend/src/components/ui/dialog.tsx:89-91` provides outer dialog motion; do not edit it. `frontend/src/index.css:84-92` is where feature keyframes live. `frontend/src/features/update-dialog.tsx:64-67` centralizes phase changes and already keeps `phaseRef`; use semantic `phase`, not raw event count, to key the inner content.

## Steps

1. Define `.update-phase-enter` in `frontend/src/index.css`: `150ms var(--ease-out)` opacity-only keyframes; reduced-motion duration `100ms`.
2. In `frontend/src/features/update-dialog.tsx`, apply a phase-keyed entrance to the noninteractive title/detail and phase-dependent body. Keep action buttons mounted and clickable according to their current conditions; do not wrap them in an animation key. Ensure an incoming `download-progress` event that leaves `phase === "downloading"` does not restart the fade.
3. Apply the entrance class only when the semantic phase changes while the dialog is open. Clear it on `animationend` so dismissing and reopening an unchanged phase does not replay it. Do not add timers or delay any action.

## Boundaries

- Do not animate percentage text, progress values, or the outer dialog again.
- Do not alter updater events, retry behavior, button availability, or API calls.
- Do not add dependencies. If cited code differs from commit `f3ce0df`, stop and reconcile before editing.

## Verification

- **Mechanical**: `pnpm --dir frontend run build` succeeds.
- **Feel check**: Exercise available → downloading → verifying → installing → ready and a failure/retry path with updater events. At 10% playback, each semantic change fades once; progress updates do not flash. Close and reopen the same release: no extra inner fade. Buttons remain clickable at once. Reduced motion uses a shorter fade with no movement.
- **Done when**: phase changes are legible without a repeated animation during download or a delayed updater action.
