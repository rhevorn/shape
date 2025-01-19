# Performance Baseline

This is an informational pre-v1 baseline, not a compatibility guarantee.
Measurements were recorded on 2026-09-03 with Go 1.27.0 on Darwin/arm64,
Apple M5 Pro, using:

```sh
go test -run '^$' -bench . -benchmem ./...
```

| Benchmark | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| String validation | 12.98 | 0 | 0 |
| Email validation | 112.4 | 88 | 5 |
| Integer bounds | 8.744 | 0 | 0 |
| Generic named number | 9.214 | 0 | 0 |
| Slice validation | 87.35 | 88 | 2 |
| Nested object validation | 252.8 | 152 | 6 |
| Invalid object validation | 926.1 | 2377 | 41 |
| Recursive validation | 335.1 | 240 | 5 |

Results vary by CPU, Go release, and toolchain settings. Future comparisons
should use the same environment and report statistically significant changes;
the project will not distort APIs to optimize synthetic benchmarks.

## Regression policy

Strict string validation, integer bounds, and generic named-number parsing are
stable hot paths and should remain at zero allocations per operation. Latency
is tracked against repeated same-machine samples, but no absolute `ns/op`
threshold is enforced across CPUs or Go releases. Collection and object
allocation counts remain observational until the v1 implementation settles.

The strict primitive entry paths perform direct type assertions. Reflection is
not used by `String`, `Int`, `Int64`, `Float64`, or `Bool`; generic numbers use
reflection only after the strict `N` assertion fails, at JSON/coercion adapter
boundaries needed to construct named numeric values.
