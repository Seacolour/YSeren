# Contributing

## Local Development

YSeren is a small Go backend with a Svelte frontend.

```bash
# backend
go run .

# frontend
cd frontend
npm install
npm run dev
```

Use `yseren.example.yaml` as the template for your local `yseren.yaml` or `yseren.yml`.
Local runtime config is intentionally ignored by git.

## Before Opening A PR

Run the same checks used during local review:

```bash
go test ./...
go vet ./...

cd frontend
npm run build
```

## Notes

- Keep changes focused and easy to review.
- Avoid committing local IDE files, binaries, or machine-specific config.
- If you change the frontend UI, please include a short description of the user-facing behavior in the PR.
