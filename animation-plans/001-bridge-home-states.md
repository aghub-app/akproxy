# 001 — Bridge home state changes

- **Status**: Implemented with Motion for React; native feel check pending. Original CSS steps superseded.
- **Commit**: f3ce0df
- **Severity**: MEDIUM
- **Category**: Missed opportunities; accessibility
- **Estimated scope**: 2 files, about 25 lines

## Problem

`frontend/src/features/home.tsx:151-172` returns three mutually exclusive page states immediately:

```tsx
if (!data) return <p className="text-sm text-muted-foreground">{t("正在读取首页…")}</p>;
if (!data.status.running) return <HomeStopped />;
```

After a user adds their first credential or starts the service, the central content appears without a visual bridge. This is occasional, meaningful state change. Do not animate initial query loading or every two-second status poll.

## Target

Give the *newly mounted* content for the `no-credentials`, `stopped`, and `running` states a single entrance. No exit wait, page navigation animation, or interaction delay. Add this token to `frontend/src/index.css` under `:root`:

```css
--ease-out: cubic-bezier(0.23, 1, 0.32, 1);
```

Add a narrowly named class in the same file:

```css
.home-state-enter {
  animation: home-state-enter 180ms var(--ease-out) both;
}
@keyframes home-state-enter {
  from { opacity: 0; transform: translateY(6px); }
  to { opacity: 1; transform: translateY(0); }
}
@media (prefers-reduced-motion: reduce) {
  .home-state-enter { animation: home-state-fade 100ms var(--ease-out) both; }
}
@keyframes home-state-fade {
  from { opacity: 0; }
  to { opacity: 1; }
}
```

Run the animation only when the semantic state changes, not when data refetches with the same state. Use a keyed wrapper or equivalent state comparison, but preserve the current subtree state while its semantic state is unchanged.

## Repo conventions to follow

`frontend/src/index.css:84-92` holds the existing `client-key-flash` keyframe; `frontend/src/features/pages.tsx:279-287` applies it only to a newly created key. Use the same conditional scope. `frontend/src/features/home.tsx:184-222` already uses Base UI tabs; keep their existing indicator motion.

## Steps

1. Add `--ease-out` and the two keyframe variants in `frontend/src/index.css`.
2. In `frontend/src/features/home.tsx`, determine one semantic state from the loaded `home.data`: `no-credentials`, `stopped`, or `running`. Apply the entrance class to the root of each of these three branches. Ensure a React key changes only when that state changes; a periodic query update must not remount `ConnectPanel`, resetting SDK selection or key reveal.
3. Keep error and loading branches instantaneous. Preserve all existing copy, routing, and API calls.

## Boundaries

- Do not animate the sidebar, tab switch, code examples, or status polling.
- Do not add dependencies or delay service controls.
- If cited branches differ from commit `f3ce0df`, stop and reconcile this plan before editing.

## Verification

- **Mechanical**: `pnpm --dir frontend run build` succeeds.
- **Feel check**: In `wails3 task dev`, add a first credential, stop/start service, and observe only the new home state entering. Refresh data without changing semantic state; no entrance repeats and current key/SDK selection remains. At 10% DevTools animation speed, confirm a 6px rise and no content overlap. Emulate reduced motion; confirm a 100ms opacity change with no movement.
- **Done when**: the three real state transitions have one brief entrance each and repeated polling has none.
