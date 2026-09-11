GO ?= go
TEST_TIMEOUT ?= 5m
TEST_ARGS ?=

.PHONY: generate test

generate:
	@for name in internal/re2go/*.re; do \
		RE_IN=$$name; \
		RE_OUT=$$(echo $$name | sed 's/\.re/.go/'); \
		re2go -W -F --input-encoding utf8 --utf8 --no-generation-date -i $$RE_IN -o $$RE_OUT; \
		gofmt -w $$RE_OUT; \
	done

test:
	$(GO) test -mod=readonly -count=1 -timeout $(TEST_TIMEOUT) $(TEST_ARGS) ./...