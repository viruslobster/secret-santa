# Name... TBD

# Development
## Install dependencies
- [Go](https://go.dev/)
- [Elm](https://guide.elm-lang.org/install/elm)
- [elm-live](https://www.elm-live.com/)
- [Task](https://taskfile.dev/)

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
