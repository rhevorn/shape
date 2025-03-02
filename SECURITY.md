# Security Policy

`shape` has no public release yet. After releases begin, the latest release on
each documented supported major line will receive security fixes. Exact support
windows will be recorded here before a stable v1 release.

Please report suspected vulnerabilities privately through
[GitHub Security Advisories](https://github.com/rhevorn/shape/security/advisories/new).
Include the affected API, required schema configuration, impact, and a minimal
reproduction when it is safe to share. Do not open a public issue before a fix
or coordinated disclosure is ready.

The standard-library-only runtime packages, bundled adapters, parsers, validation resource
bounds, generated JSON Schema/OpenAPI semantics, and concurrency guarantees are
in scope. Security properties of caller callbacks, application authorization,
server timeouts, rate limits, and downstream schema consumers remain the
application's responsibility.
