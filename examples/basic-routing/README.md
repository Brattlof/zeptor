# Basic Routing

Demonstrates all Zeptor routing features:
- Static routes (`/`, `/about`)
- Dynamic routes (`/{slug}`)
- API endpoints (`/api/users`)

## Usage

```bash
cd examples/basic-routing
zt dev
```

## Routes

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Home page |
| GET | `/about` | About page |
| GET | `/{slug}` | Dynamic route (e.g., `/hello-world`) |
| GET, POST | `/api/users` | Users API |
| GET | `/api/routes` | List all routes |
