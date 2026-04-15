# Cmux Frontend

A production-ready React project template with a modern tech stack, full testing infrastructure, and Docker support.

## Generate a New Project

```bash
# Clone the template
git clone <repo-url> cmux-frontend
cd cmux-frontend

# Generate your project
./bin/create.sh my-cool-app

# Or specify a target directory
./bin/create.sh my-cool-app ~/projects/my-cool-app

# Skip npm install (faster, do it yourself later)
./bin/create.sh my-cool-app --no-install

# Skip git init
./bin/create.sh my-cool-app --no-git
```

The generator will:

- Copy all template files to a new directory
- Replace the project name everywhere (package.json, Makefile, env files, README, HTML title, etc.)
- Derive a human-readable title from the kebab-case name (`my-cool-app` → `My Cool App`)
- Initialize a git repo with an initial commit
- Install npm dependencies

## Tech Stack

| Layer            | Choice                                        |
| ---------------- | --------------------------------------------- |
| Framework        | Vite 6 + React 19 + SWC + TypeScript (strict) |
| Styling          | Tailwind CSS 4 (CSS-native `@theme`)          |
| Routing          | React Router 7 (declarative/library mode)     |
| Server State     | TanStack Query 5                              |
| Client State     | Zustand 5 (slices pattern)                    |
| Unit/Integration | Vitest + Testing Library + MSW 2              |
| E2E              | Playwright (Chromium, Firefox, WebKit)        |
| Linting          | ESLint 9 (flat config) + Prettier             |
| Git Hooks        | Husky + lint-staged                           |
| Container        | Multi-stage Docker + nginx                    |

## Getting Started

```bash
# Install dependencies
make install

# Start dev server (http://localhost:5173)
make dev

# Or use Docker for dev
make docker-dev
```

## Available Commands

| Command             | Description                          |
| ------------------- | ------------------------------------ |
| `make install`      | Install dependencies (`npm ci`)      |
| `make dev`          | Start Vite dev server                |
| `make build`        | Production build                     |
| `make preview`      | Preview production build             |
| `make lint`         | Run ESLint                           |
| `make format`       | Format code with Prettier            |
| `make format-check` | Check formatting                     |
| `make typecheck`    | Run TypeScript compiler check        |
| `make test`         | Run Vitest in watch mode             |
| `make test-run`     | Run Vitest once (CI mode)            |
| `make test-e2e`     | Run Playwright E2E tests             |
| `make docker-dev`   | Start Docker dev environment         |
| `make docker-build` | Build production Docker image        |
| `make docker-run`   | Run production container (port 8080) |
| `make clean`        | Remove generated files               |

## Project Structure

```
src/
├── main.tsx                  # Entry point
├── App.tsx                   # Root component (providers)
├── index.css                 # Tailwind v4 @theme config
├── config/                   # App configuration
│   ├── env.ts                # Typed env vars with validation
│   ├── query-client.ts       # TanStack Query client
│   └── feature-flags.ts      # Env-driven feature flags
├── components/
│   ├── ui/                   # Reusable UI components (Button)
│   ├── layout/               # RootLayout, PageLayout
│   └── common/               # ErrorBoundary, LoadingSpinner
├── features/example/         # Feature module pattern
│   ├── components/           # Feature-specific components
│   ├── hooks/                # TanStack Query hooks
│   ├── services/             # API service functions
│   ├── stores/               # Feature Zustand store
│   └── types/                # Feature types
├── hooks/                    # Shared hooks (useMediaQuery)
├── lib/                      # Utilities (api-client, cn)
├── pages/                    # Page components
├── router/                   # React Router config
├── stores/                   # Global Zustand store (slices)
├── test/                     # Test utilities, MSW mocks
└── types/                    # Shared type definitions
```

## Architecture

### Feature Modules

Each feature follows a vertical slice pattern under `src/features/`:

```
features/my-feature/
├── components/       # UI components
├── hooks/            # Data-fetching hooks (TanStack Query)
├── services/         # API client functions
├── stores/           # Local Zustand store
├── types/            # TypeScript types
└── index.ts          # Public API (barrel export)
```

### State Management

- **Server state**: TanStack Query handles caching, background refetches, and optimistic updates
- **Client state**: Zustand with the slices pattern — each slice is self-contained in `stores/slices/`, combined in `stores/app.store.ts`

### Styling

Tailwind CSS v4 with CSS-native `@theme` in `src/index.css`. Custom design tokens (colors, fonts) are defined there. Use the `cn()` utility from `src/lib/cn.ts` for conditional class merging.

### Testing

- **Unit/Integration**: Vitest + Testing Library with MSW for API mocking. Custom `render()` from `src/test/test-utils.tsx` wraps components in all providers.
- **E2E**: Playwright with multi-browser support. Tests in `e2e/`.

### API Client

Fetch-based typed client in `src/lib/api-client.ts`. No external HTTP dependencies. Base URL configured via `VITE_API_BASE_URL` env var.

## Environment Variables

Copy `.env.example` to `.env` and configure:

| Variable            | Description       | Default                     |
| ------------------- | ----------------- | --------------------------- |
| `VITE_API_BASE_URL` | API base URL      | `http://localhost:2689/api` |
| `VITE_APP_TITLE`    | Application title | Project title               |

## Docker

### Development

```bash
make docker-dev
```

Starts the dev server in a container with hot-reload via volume mounts.

### Production

```bash
make docker-build
make docker-run
```

Builds a multi-stage image (Node for building, nginx for serving). The production container runs nginx as a non-root user on port 8080.

## License

MIT
