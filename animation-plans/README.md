# Animation plans

These plans cover all five opportunities selected from the read-only page sweep. They were written against commit `f3ce0df`. The implementation now uses Motion for React, following the user's later direction; the original CSS implementation steps in the plans are superseded. Code and build verification are complete. Native-window feel checks remain pending because the desktop automation channel could not read the window.

| Order | Plan | Severity | Status | Dependency |
| --- | --- | --- | --- | --- |
| 1 | [001 — Bridge home state changes](001-bridge-home-states.md) | MEDIUM | Implemented, feel check pending | — |
| 2 | [002 — Reveal the first provider account](002-reveal-first-account.md) | LOW | Implemented, feel check pending | 001 |
| 3 | [003 — Reveal the custom listen address](003-reveal-custom-listen-address.md) | LOW | Implemented, feel check pending | 001 |
| 4 | [004 — Reveal the custom update interval](004-reveal-custom-update-interval.md) | LOW | Implemented, feel check pending | 003 |
| 5 | [005 — Bridge update dialog phase content](005-bridge-update-dialog-phases.md) | MEDIUM | Implemented, feel check pending | 001 |

Execute 001 first to establish the shared `--ease-out` curve. 002, 003, and 005 can then be taken independently; 004 reuses 003's class. Review feel checks in the running Wails app before marking any plan done.
