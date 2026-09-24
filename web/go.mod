// This file exists only to mark web/ as a separate Go module boundary, so
// `go build ./...`/`go test ./...` from the repo root do not descend into
// web/node_modules and pick up stray .go files some npm package happens to
// ship (e.g. flatted's Go bindings). web/ is a Next.js app; nothing here
// builds or is depended on.
module github.com/anshiq/ideserver/web-nogo

go 1.23
