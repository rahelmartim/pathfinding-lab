SITE := site
PORT ?= 8080

.PHONY: test bench web serve pathfind hanoi clean

## test: vet and run the Go tests with the race detector
test:
	go vet ./...
	go test -race ./...

## bench: the Go benchmarks of the algorithms
bench:
	go test -run x -bench . ./search

## web: build the page into site/
web:
	rm -rf $(SITE) && mkdir -p $(SITE)
	cp web/index.html web/style.css web/app.js web/view.js $(SITE)/
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" $(SITE)/
	GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o $(SITE)/lab.wasm ./cmd/lab-wasm

## serve: build the page and serve it on http://localhost:$(PORT)
serve: web
	python3 -m http.server $(PORT) --directory $(SITE)

## pathfind: every algorithm on a random map, in the terminal
pathfind:
	go run ./cmd/pathfind

## hanoi: the Tower of Hanoi with BFS, in the terminal
hanoi:
	go run ./cmd/hanoi

clean:
	rm -rf $(SITE)
