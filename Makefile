# libreadme. `make` runs the checks; `make build` puts the command in bin/,
# which is gitignored; `make smoke` runs it against the corvid theme in
# testdata, which passes, and then against a sheet with the vaporwave web
# pink declared as text, which fails at 7.4:1 under a 10:1 floor, so the
# gate is seen doing both of its jobs.
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
	@printf ':root {\n  --ground: #160d2b;\n  --pink: #ff6ec7;\n}\n' > bin/pink-as-text.css
	@./bin/libreadme check bin/pink-as-text.css -text pink -surfaces ground || true
