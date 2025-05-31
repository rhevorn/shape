# Release preparation

The first release uses the API described in [API.md](API.md). Pre-release code
must follow the [migration notes](USAGE.md#9-migrating-pre-release-code):
`FromTags` replaces `Struct`, pointer presence uses `NotNull`, container
transforms precede element transforms, and `And` follows declaration order.

## Candidate checks

Run from the repository root:

```sh
make release-check
GOTOOLCHAIN=go1.24.0 go test -race ./...
cd tools/shapevet
GOTOOLCHAIN=go1.24.0 go test -race ./...
```

`release-check` includes formatting, vet, root and analyzer tests, candidate
module isolation, shapevet self-check, race detection, fuzz smoke, benchmarks,
and runnable examples. CI runs the main checks on Go 1.24 and the current stable
Go release; examples are included in the CI gate.

`shapevet-isolated` uses a temporary modfile and `GOWORK=off`. It checks the
candidate root module and builds/tests the tool against that root without
depending on a tag that has not been published yet. It does not edit the
committed module files and is not a substitute for testing the published module.

For a performance comparison, use the same machine and Go version:

```sh
go test -run '^$' -bench 'Benchmark(NoopPayload|NestedJSON|SchemaParseJSON)' -benchmem -count=5 .
```

The regression suite covers mutable error/export data, declaration order,
pointer and collection defaults, canceled contexts, typed map keys, mixed
transform paths, finite checks after traversal pruning, export intersections,
unsupported wire representations, and recursive analyzer inputs.

## Export boundary

Export is currently a tagged-schema adapter. Explicit `New` Schemas remain
runtime-only. Byte slices, `json:",string"`, custom codecs, Go duration strings,
fallbacks, custom callbacks and unsupported regexps return
`UnsupportedSchemaError`. Multiple exportable rules intersect rather than
overwriting one another. Only the supported Go regexp subset is translated to
ECMA-262; consumers must enable format assertions if relying on format rules.

The runtime library and adapters have no third-party runtime dependencies.
Independent conformance tools used during a release should remain outside the
library module graph.

## Publish order

1. Choose the release version, finish the changelog, and ensure the version
   required by `tools/shapevet/go.mod` matches the intended root release.
2. Run the candidate checks and review the final diff. Commit the candidate.
3. Publish the root module tag first (the existing tool requirement is
   `v0.1.0`; change it if choosing a different version).
4. In `tools/shapevet`, with no local replacement, run:

   ```sh
   GOWORK=off go mod download
   GOWORK=off go mod verify
   GOWORK=off go test ./...
   GOWORK=off go vet ./...
   ```

5. Publish the matching nested-module tag, for example `tools/shapevet/v0.1.0`.
   Check installation of both published modules in a fresh consumer project.
