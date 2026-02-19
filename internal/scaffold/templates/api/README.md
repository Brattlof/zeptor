# {{.ProjectName}}

A Zeptor API application.

## Development

```bash
zt dev
```

Open http://localhost:{{.Port}}

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/hello` | Hello message |
| GET | `/api/users` | List users |

## Example

```bash
curl http://localhost:{{.Port}}/api/hello
curl http://localhost:{{.Port}}/api/users
```

## Build

```bash
zt build
```

## Learn More

- [Zeptor Documentation](https://github.com/brattlof/zeptor)
