# libreadme. the same verbs as libtheme-css, so the two read the same:
#   make              gofmt, go vet, go test
#   make build        every command under cmd/ into bin/, which is gitignored
#   make clean        bin/ away again
#   make visualtest   the command seen doing its job: corvid passes, and a
#                     sheet with the vaporwave web pink declared as text
#                     fails at 7.4:1 under a 10:1 floor
#   make docs         what go doc extracts from the comments
.PHONY: all fmt vet test build clean visualtest docs
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
visualtest: build
	@./bin/libreadme check testdata/corvid.css -text ink,dim -surfaces ground,pane -chips green,red,accent || true
	@printf ':root {\n  --ground: #160d2b;\n  --pink: #ff6ec7;\n}\n' > bin/pink-as-text.css
	@./bin/libreadme check bin/pink-as-text.css -text pink -surfaces ground || true
docs:
	@go doc -all ./profile
	@go doc -all ./colour
	@go doc -all ./check
	@go doc -all ./css
