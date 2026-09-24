# 004 — Reveal the custom update interval

- **Status**: Implemented with Motion for React; native feel check pending. Original CSS steps superseded.
- **Commit**: f3ce0df
- **Severity**: LOW
- **Category**: Missed opportunities; state indication
- **Estimated scope**: 1 file, about 30 lines
- **Depends on**: 003 for `.custom-field`

## Problem

In Settings → About, `frontend/src/features/about.tsx:214-233` mounts a number field immediately when the user selects a custom check interval:

```tsx
{preset === "custom" ? (
  <NumberField
    value={hours}
    min={1}
    max={720}
    step={1}
    disabled={disabled}
```

The control is a sibling of the interval selector, so the new field can appear abruptly.

## Target

Use the exact `.custom-field` from plan 003: opacity `0 → 1` and `translateY(-4px) → 0` on entry, reverse on exit, `160ms cubic-bezier(0.23, 1, 0.32, 1)`. Reduced motion uses a `100ms` fade only. Disable the numeric field immediately when another preset is chosen, then remove it after the exit.

## Repo conventions to follow

The conditional `NumberField` is already controlled by `preset` (`frontend/src/features/about.tsx:180-182`). `frontend/src/features/pages.tsx:420-426`, after plan 003, is the matching settings pattern. No new token or keyframe is needed.

## Steps

1. Wrap the conditionally rendered `NumberField` in `frontend/src/features/about.tsx:214-233` with `.custom-field` and the same entering/closing presence logic from plan 003. On closing, make its wrapper `inert` and `aria-hidden` and disable `NumberField` immediately; unmount after `transitionend` or the 160ms fallback timer. Cancel removal on rapid reversal.
2. Preserve the existing `NumberFieldGroup` width and numeric validation. The current `disabled` prop also reflects automatic-check preferences; combine it with the closing state instead of replacing it.

## Boundaries

- Do not animate update checking, the switch controls, or preset changes that do not mount the number field.
- Do not change preference persistence or add dependencies.
- If the cited code or plan 003 class differs from commit `f3ce0df`, stop and reconcile before editing.

## Verification

- **Mechanical**: `pnpm --dir frontend run build` succeeds.
- **Feel check**: Select Custom, a fixed interval, then Custom again quickly. Entry and exit follow the same 4px path and retarget smoothly; the closing control cannot receive focus. At 10% playback there is no height/width animation; reduced motion fades without movement.
- **Done when**: the custom interval transition matches the listen-address transition exactly.
