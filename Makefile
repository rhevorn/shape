.PHONY: bench check examples-check fmt fmt-check fuzz-smoke release-check shapevet-check shapevet-isolated shapevet-test test test-race vet

check: fmt-check vet test

shapevet-test:
	cd tools/shapevet && go test ./...

shapevet-isolated:
	sh scripts/check-isolated.sh

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
	@tool_dir=$$(mktemp -d); trap 'rm -rf "$$tool_dir"' EXIT HUP INT TERM; \
	go build -o "$$tool_dir/shapevet" ./tools/shapevet && \
	go vet -vettool="$$tool_dir/shapevet" -exclude-tests ./...

fuzz-smoke:
	go test ./transform -run '^$$' -fuzz '^FuzzStringTransformer$$' -fuzztime 2s
	go test ./types -run '^$$' -fuzz '^FuzzDurationJSON$$' -fuzztime 2s
	go test . -run '^$$' -fuzz '^FuzzTaggedSchemaJSON$$' -fuzztime 2s
	go test . -run '^$$' -fuzz '^FuzzParameterDecoding$$' -fuzztime 2s
	go test . -run '^$$' -fuzz '^FuzzBindRequest$$' -fuzztime 2s

bench:
	go test -run '^$$' -bench . -benchmem ./...

examples-check:
	@for dir in $$(rg --files examples -g 'main.go' | sed 's|/main.go$$||' | sort); do \
		go run ./$$dir >/dev/null || exit 1; \
	done

release-check: check shapevet-test shapevet-isolated shapevet-check test-race fuzz-smoke bench examples-check
