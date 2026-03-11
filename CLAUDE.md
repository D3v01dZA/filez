# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is Filez

A single-binary Go file sharing server. Files can be uploaded once and downloaded once — downloading a file automatically deletes it.

## Build & Run

```sh
./build.sh        # builds ./filez binary
./run.sh          # builds and runs the server (passes args to filez)
go build -o filez .
```

The server listens on `:8080` by default. Configure with env vars:
- `PORT` — listen port (default `8080`)
- `STORAGE_DIR` — where uploaded files are stored (default `./data`)

## Docker

```sh
docker buildx build -t filez .
# or use docker-compose.yml
```

`stable.sh` builds and pushes multi-arch image to `d3v01d/filez:stable`.

## Architecture

Single-file Go server (`main.go`) with an embedded HTML frontend (`index.html` via `//go:embed`).

- **Storage**: uploaded files are saved to `STORAGE_DIR` with random hex IDs. A `files.json` metadata file in the same directory maps IDs to original filenames.
- **State**: in-memory `map[string]fileEntry` protected by a `sync.Mutex`, persisted to `files.json` on every upload/download.
- **Download-and-delete**: downloading a file removes it from both the map and disk atomically within the same request handler.

### HTTP Endpoints

| Route | Method | Purpose |
|---|---|---|
| `/` | GET | Serves embedded `index.html` |
| `/upload` | POST | Multipart file upload, returns JSON `{name, id}` |
| `/files` | GET | Lists available files as JSON array |
| `/download/{id}` | GET | Downloads and deletes file |

No external dependencies — stdlib only (`go.mod` has no requires).
