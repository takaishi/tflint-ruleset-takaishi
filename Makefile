.PHONY: build
build:
	go build -o dist/tflint-ruleset-takaishi .

.PHONY: install
install: build
	mkdir -p ~/.tflint.d/plugins/
	cp dist/tflint-ruleset-takaishi ~/.tflint.d/plugins/tflint-ruleset-takaishi
	chmod +x ~/.tflint.d/plugins/tflint-ruleset-takaishi

.PHONY: test
test:
	go test ./...
