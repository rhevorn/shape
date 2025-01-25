# Security Policy

GoShape has no public release yet. The first supported release will be v1.0.0;
after that, the latest v1.x release receives security fixes.

Please report suspected vulnerabilities privately through
[GitHub Security Advisories](https://github.com/rhevorn/goshape/security/advisories/new).
Include the affected API, required schema configuration, impact, and a minimal
reproduction when it is safe to share. Do not open a public issue before a fix
or coordinated disclosure is ready.

The dependency-free root module, bundled adapters, parsers, validation resource
bounds, generated JSON Schema/OpenAPI semantics, and concurrency guarantees are
in scope. Security properties of caller callbacks, application authorization,
server timeouts, rate limits, and downstream schema consumers remain the
application's responsibility.
