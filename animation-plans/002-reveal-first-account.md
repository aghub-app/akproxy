# 002 — Reveal the first provider account

- **Status**: Implemented with Motion for React; native feel check pending. Original CSS steps superseded.
- **Commit**: f3ce0df
- **Severity**: LOW
- **Category**: Missed opportunities; accessibility
- **Estimated scope**: 2 files, about 20 lines
- **Depends on**: 001 for `--ease-out`

## Problem

All six provider routes use `AccountsPanel` (`frontend/src/features/pages.tsx:506-535`). In `frontend/src/features/accounts.tsx:274-285`, the first successful login replaces `ProviderEmpty` with a grid instantly:

```tsx
if (accounts.data.length === 0) {
  return <ProviderEmpty page={page}>{actions}</ProviderEmpty>;
}
return (
  <section className="flex flex-col gap-3">
    <div className="flex flex-wrap gap-2">{actions}</div>
    <ul className="grid grid-cols-2 items-start gap-3">
      {accounts.data.map((account) => (
```

This rare transition should clarify that the account was added. Animating periodic quota updates or every account card would hinder reading.

## Target

Only a card created by a successful login enters from `opacity: 0; transform: scale(.97)` to normal over `200ms var(--ease-out)` (`cubic-bezier(0.23, 1, 0.32, 1)`). Transform origin is the card center. Under `prefers-reduced-motion: reduce`, use opacity only over `100ms`. No stagger; cards stay interactive throughout.

## Repo conventions to follow

`frontend/src/features/pages.tsx:279-287` marks only a newly created key with `flashed` state and clears it on `animationend`. Follow that event-driven pattern to avoid replaying motion on the five-second account refetch (`frontend/src/features/accounts.tsx:171-177`). Put keyframes in `frontend/src/index.css`, alongside `client-key-flash` at line 84. Use the shared `--ease-out` from plan 001.

## Steps

1. In `frontend/src/features/accounts.tsx`, track account IDs that arrive after a successful login or reauthorization while this panel is mounted. Compare previous and next account IDs. Mark only genuinely new IDs; do not mark an initially loaded list or every refetch. The first account after an empty state must be covered.
2. Give only marked `<li>` elements an entrance class. Clear the mark on `animationend` so a later quota render does not replay it. Do not reset the state on unrelated status or preference updates.
3. In `frontend/src/index.css`, define a `200ms var(--ease-out)` opacity/scale entrance and a `100ms` fade-only reduced-motion variant.

## Boundaries

- Covers Codex, Grok, Claude, Gemini, Kimi, and Devin through their shared panel. No per-provider copies.
- Do not animate quota bars, deletion, authorization alerts, or background polling.
- Do not add dependencies or change account fetch logic.
- If cited code differs from commit `f3ce0df`, stop and reconcile before editing.

## Verification

- **Mechanical**: `pnpm --dir frontend run build` succeeds.
- **Feel check**: On one empty provider page, add an account; see a single 3% scale entrance. Switch routes and return, then wait for refresh; existing cards stay still. Add a second account; only it enters. At 10% playback, confirm the card grows from its center. Reduced motion has a brief fade and no scale.
- **Done when**: new cards enter once, existing cards never replay, and all six provider routes share the behavior.
