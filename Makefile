# libreadme. `make` runs the checks; `make build` puts the command in bin/,
# which is gitignored; `make smoke` runs it against the theme that shipped
# with a glare ink, which it should catch.
.PHONY: all fmt vet test build clean smoke
all: fmt vet test
fmt:
	gofmt -l -w .
vet:
	go vet ./...
test:
	go test ./...
build:
	@mkdir -p bin
	@go build -o bin/ ./cmd/...
clean:
	rm -rf bin
smoke: build
	@./bin/libreadme check testdata/corvid.css -text ink,dim -surfaces ground,pane -chips green,red,accent || true
