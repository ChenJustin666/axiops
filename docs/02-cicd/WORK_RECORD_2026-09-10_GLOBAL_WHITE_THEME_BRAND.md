# 2026-09-10 Global White Theme and AxiOps Brand Refresh

## Completed

- Fixed theme bootstrap so the browser operating-system dark preference cannot
  reapply a dark AxiOps shell. The global baseline is now LIGHT.
- Applied white backgrounds and dark typography to the global application shell,
  sidebar, top header, menu items, project selector, cluster pages and CI/CD
  service variable panel.
- Corrected the CI/CD `components and variables` panel so headings, values,
  labels and Element Plus inputs use readable dark text on white surfaces.
- Refreshed the shared `ArtLogo` with an AxiOps teal/blue geometric mark.
- Refreshed project icons for YAML, Helm, host and workspace views with a
  consistent professional gradient palette.

## Verification

- Prettier: passed for all changed files.
- Targeted ESLint: passed for theme, layout, logo and project files.
- Full `vue-tsc`: remains blocked by pre-existing repository-wide type errors;
  no new error was reported from the changed theme/logo files.
- `npm.cmd run build:dev`: blocked by Windows `esbuild` `spawn EPERM` policy;
  this is an environment execution restriction, not a reported source error.

## Operations

- No SQL changes are required.
- No backend restart is required.
- Development mode: refresh the browser or rely on Vite HMR.
- Deployed frontend: rebuild and restart the frontend process to publish the
  updated assets.
