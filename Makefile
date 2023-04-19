test-all:
	@echo "Test normal regex"
	@echo
	go test -timeout 30s ./...
	@echo
	@echo "Test re2 WASM regex"
	go test -tags re2_wasm -timeout 30s ./...
	@echo
	@echo "Test re2 cgo regex"
	go test -tags re2_cgo -timeout 30s ./...