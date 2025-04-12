.PHONY: bench check fmt fmt-check fuzz-smoke release-check shapevet-check shapevet-isolated shapevet-test test test-race vet

check: fmt-check vet test

shapevet-test:
	cd tools/shapevet && go test ./...

shapevet-isolated:
	cd tools/shapevet && GOWORK=off go mod download
	cd tools/shapevet && GOWORK=off go mod verify
	cd tools/shapevet && GOWORK=off go test ./...
	cd tools/shapevet && GOWORK=off go vet ./...

fmt:
	gofmt -w $$(rg --files -g '*.go')

fmt-check:
	@test -z "$$(gofmt -l .)" || { gofmt -d .; exit 1; }

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

shapevet-check:
	cd tools/shapevet && go build -o /tmp/shapevet-vettool .
	go vet -vettool=/tmp/shapevet-vettool -exclude-tests ./...

fuzz-smoke:
	go test ./transform -run '^$$' -fuzz '^FuzzStringTransformer$$' -fuzztime 2s
	go test ./types -run '^$$' -fuzz '^FuzzDurationJSON$$' -fuzztime 2s
	go test . -run '^$$' -fuzz '^FuzzTaggedSchemaJSON$$' -fuzztime 2s

bench:
	go test -run '^$$' -bench . -benchmem ./...

release-check: check shapevet-test shapevet-isolated shapevet-check test-race fuzz-smoke bench
