# 2026-09-10 Project Workspace Entry Alignment

## Problem

- Existing configured projects such as `fghfgh` opened the service workbench,
  while newly created projects were redirected to the empty overview tab.
- Workspace tab state was not represented in the URL.
- Reusing the workspace component for another project ID could retain stale
  project state because the route parameter was captured only once.

## Completed

- New projects now enter the same `service configuration` workbench used by
  existing projects through `?tab=services&setup=1`.
- Workspace tab selection is stored in the URL and restored after refresh.
- Switching project IDs resets project-scoped services, environments, clusters,
  workflows and selections before loading the next project.
- Corrected the run-list frontend type to match the build-run API response.

## Verification

- Prettier: passed.
- Targeted ESLint: passed.
- Targeted Vue type check: no errors for the two changed project pages.
- Local frontend responds with HTTP 200 on `http://127.0.0.1:3006`.

## Operations

- No SQL changes are required.
- No backend restart is required.
- The local Vite frontend is already running and will hot reload the change;
  refresh the browser before testing.
