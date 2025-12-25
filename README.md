# Name... TBD

# Development
## Install dependencies
- [Go](https://go.dev/)
- [Elm](https://guide.elm-lang.org/install/elm)
- [Task](https://taskfile.dev/)

## Build everything and run the server
```
❯ task
task: [default] task frontend
task: [frontend] elm make www/HomePage.elm --output elm.js
Success!     

    HomePage ───> elm.js

task: [default] task backend
task: [backend] go build
task: [default] ./secret-santa
2025/12/25 13:52:54 Starting server on port 8080...
2025/12/25 13:52:54 Serving files from current directory
```
