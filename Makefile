.PHONY: bench check fmt fuzz-smoke release-check test test-race vet

check: fmt vet test

fmt:
	@test -z "$$(gofmt -l .)" || { gofmt -d .; exit 1; }

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

fuzz-smoke:
	go test -run '^$$' -fuzz '^FuzzStringSchema$$' -fuzztime 2s
	go test -run '^$$' -fuzz '^FuzzIntJSON$$' -fuzztime 2s
	go test -run '^$$' -fuzz '^FuzzGenericNumberJSON$$' -fuzztime 2s
	go test -run '^$$' -fuzz '^FuzzObject$$' -fuzztime 2s
	go test -run '^$$' -fuzz '^FuzzParse$$' -fuzztime 2s
	go test -run '^$$' -fuzz '^FuzzTupleJSON$$' -fuzztime 2s
	go test -run '^$$' -fuzz '^FuzzRecordJSON$$' -fuzztime 2s
	go test -run '^$$' -fuzz '^FuzzLazyJSON$$' -fuzztime 2s
	go test -run '^$$' -fuzz '^FuzzJSONSchemaExport$$' -fuzztime 2s
	go test -run '^$$' -fuzz '^FuzzParseReaderLimit$$' -fuzztime 2s

bench:
	go test -run '^$$' -bench . -benchmem ./...

release-check: check test-race fuzz-smoke bench
