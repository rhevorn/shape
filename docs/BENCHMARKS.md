# Performance Baseline

This is an informational pre-v1 baseline, not a compatibility guarantee.
Measurements were recorded on 2026-09-03 with Go 1.27.0 on Darwin/arm64,
Apple M5 Pro, using:

```sh
go test -run '^$' -bench . -benchmem ./...
```

| Benchmark | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| String validation | 12.78 | 0 | 0 |
| Email validation | 118.3 | 88 | 5 |
| Integer bounds | 9.035 | 0 | 0 |
| Generic named number | 9.142 | 0 | 0 |
| Slice validation | 85.01 | 88 | 2 |
| Nested object validation | 248.3 | 152 | 6 |
| Invalid object validation | 923.7 | 2377 | 41 |
| Recursive validation with depth accounting | 823.7 | 496 | 9 |
| 1,000 distinct comparable values with `Unique` | 38,087 | 68,824 | 752 |
| Reject compact huge integer exponent | 40.01 | 136 | 3 |

Results vary by CPU, Go release, and toolchain settings. Future comparisons
should use the same environment and report statistically significant changes;
the project will not distort APIs to optimize synthetic benchmarks.

## Regression policy

Strict string validation, integer bounds, and generic named-number parsing are
stable hot paths and should remain at zero allocations per operation. Latency
is tracked against repeated same-machine samples, but no absolute `ns/op`
threshold is enforced across CPUs or Go releases. Collection and object
allocation counts remain observational until the v1 implementation settles.
Recursive parsing deliberately pays for synchronized parse-local depth state;
the bound prevents cyclic values and deep error paths from exhausting process
resources. Comparable uniqueness is linear in element count; its interface-key
map allocations are tracked as an optimization opportunity, not an API gate.

The strict primitive entry paths perform direct type assertions. Reflection is
not used by `String`, `Int`, `Int64`, `Float64`, or `Bool`; generic numbers use
reflection only after the strict `N` assertion fails, at JSON/coercion adapter
boundaries needed to construct named numeric values.
