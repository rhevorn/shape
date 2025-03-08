.PHONY: bench check fmt fuzz-smoke release-check shapevet-test test test-race vet

check: fmt vet test

shapevet-test:
	go test ./tools/shapevet/...

fmt:
	@test -z "$$(gofmt -l .)" || { gofmt -d .; exit 1; }

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

fuzz-smoke:
	go test ./transform -run '^$$' -fuzz '^FuzzStringTransformer$$' -fuzztime 2s
	go test ./types -run '^$$' -fuzz '^FuzzDurationJSON$$' -fuzztime 2s
	go test . -run '^$$' -fuzz '^FuzzTaggedSchemaJSON$$' -fuzztime 2s

bench:
	go test -run '^$$' -bench . -benchmem ./...

release-check: check shapevet-test test-race fuzz-smoke bench
