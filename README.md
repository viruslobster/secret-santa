# Name... TBD

# Development
## Install dependencies
- [Go](https://go.dev/)
- [Elm](https://guide.elm-lang.org/install/elm)
- [elm-live](https://www.elm-live.com/)
- [Task](https://taskfile.dev/)

## Database
SQLite uses `github.com/mattn/go-sqlite3`, which requires CGO enabled and a C compiler.

Run migrations before starting the server:
```sh
go run ./cmd/hoho migrate
```
Or install the CLI with `go install ./cmd/hoho`, then run `hoho migrate`.
Both the server and CLI use `./db.sqlite` relative to the working directory.
Add numbered SQL files to `migrations/`; they run in filename order and are
tracked in `_migrations`. Migrations are embedded, so rebuild the CLI after adding files.

## Build everything, run the server, automatically rebuild frontend changes
```
❯ task
task: [serve] ./secret-santa
task: [watch] elm-live www/Main.elm -- --output=dist/main.js --debug
task: [backend] go build
2025/12/26 18:06:18 Starting server on port 8080...
2025/12/26 18:06:18 Serving files from current directory
task: [frontend] mkdir -p dist
task: [frontend] elm make www/Main.elm --output dist/main.js
Success!     

    Main ───> dist/main.js


elm-live:
  Server has been started! Server details below:
    - Website URL: http://localhost:8000
    - Serving files from: /home/viruslobster/src/secret-santa
  

elm-live:
  The build has succeeded. 

elm-live:
  Watching the following files:
    - www/**/*.elm
```
