# 003 — Reveal the custom listen address

- **Status**: Implemented with Motion for React; native feel check pending. Original CSS steps superseded.
- **Commit**: f3ce0df
- **Severity**: LOW
- **Category**: Missed opportunities; state indication
- **Estimated scope**: 2 files, about 35 lines
- **Depends on**: 001 for `--ease-out`

## Problem

In Settings → Listen, `frontend/src/features/pages.tsx:420-426` mounts the custom address input immediately when the select changes:

```tsx
{form.listenMode === "custom" ? (
  <Input
    value={form.customHost}
    placeholder="192.168.1.8"
    onChange={(event) => edit({ customHost: event.target.value })}
  />
) : null}
```

An occasional choice reveals a new required field. A short entrance makes the conditional relationship legible.

## Target

Enter from `opacity: 0; transform: translateY(-4px)` to normal over `160ms var(--ease-out)` (`cubic-bezier(0.23, 1, 0.32, 1)`). Exit along the same 4px path over `160ms var(--ease-out)`, then unmount. Disable and remove the field from tab order as soon as the user deselects Custom. Reduced motion: opacity only, `100ms`. Do not animate the dropdown itself; Base UI handles it.

## Repo conventions to follow

`frontend/src/index.css:84-92` stores app-specific keyframes. `frontend/src/features/pages.tsx:279-287` shows how feature markup opts into a narrowly scoped animation. The select remains controlled through `edit({ listenMode: value }, true)` at `frontend/src/features/pages.tsx:400-407`.

## Steps

1. Add `.custom-field` in `frontend/src/index.css` with `transition: opacity 160ms var(--ease-out), transform 160ms var(--ease-out)`, normal state `opacity: 1; transform: translateY(0)`, and closing state `opacity: 0; transform: translateY(-4px)`. Add `@starting-style` with the same closing values for entry. Under `prefers-reduced-motion: reduce`, use `transform: none` in every state and `opacity 100ms var(--ease-out)`.
2. In `frontend/src/features/pages.tsx:420-426`, keep the field mounted during the 160ms exit with a local presence flag. On deselection, mark it closing, set `disabled`, `aria-hidden`, and `inert` on its wrapper immediately, then remove it after the transition. Cancel the pending removal if Custom is chosen again. On entry, it must be enabled and available without waiting for animation. Keep the existing `edit` call and validation unchanged.
3. At `transitionend`, complete removal; include a 160ms timeout fallback for environments that suppress transition events. Clear the timer on unmount. Do not animate height or width.

## Boundaries

- Do not animate port edits, tab navigation, or the select popup.
- Do not add dependencies or change validation and persistence.
- If cited code differs from commit `f3ce0df`, stop and reconcile before editing.

## Verification

- **Mechanical**: `pnpm --dir frontend run build` succeeds.
- **Feel check**: Select custom, local, custom rapidly. The entrance and exit follow the same 4px path, rapid reversal does not snap or replay from zero, and the closing input cannot receive focus. At 10% playback, confirm no animation of height or width. Reduced motion shows only a 100ms fade.
- **Done when**: revealing and hiding the custom address are legible, and an inactive input is disabled immediately.
