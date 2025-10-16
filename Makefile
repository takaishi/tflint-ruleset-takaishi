.PHONY: build
build:
	go build -o dist/tflint-ruleset-takaishi ./cmd/tflint-ruleset-takaishi

.PHONY: install
install: build
	mkdir -p ~/.tflint.d/plugins/
	cp dist/$(PLUGIN_NAME) ~/.tflint.d/plugins/$(PLUGIN_NAME)
	chmod +x ~/.tflint.d/plugins/$(PLUGIN_NAME)

.PHONY: test
test:
	go test -race ./...
