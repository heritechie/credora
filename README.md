# Credora

Open-source credit decisioning infrastructure.

> Your providers. Your keys. Your decisions.

Credora is an open-source infrastructure layer for orchestrating credit assessment workflows using customer-selected providers, deterministic policies, scoring, evidence, and audit trails.

Credora is NOT:

- a lender
- a credit bureau
- a KYC provider
- a fraud provider
- a proprietary scoring provider

## Project Structure

```text
apps/
  landing/          Astro landing page
services/
  engine/           Go decisioning engine
examples/
  personal-loan/    First example assessment (coming soon)
deploy/             Deployment and infrastructure
docs/               Architecture and developer documentation
```

## Getting Started

```bash
make setup
make dev
```

See `docs/` for architecture and developer documentation.

## API Documentation

Run the engine:

```bash
cd services/engine
go run ./cmd/server
```

Then open:

http://localhost:8080/docs

The interactive documentation is powered by [Scalar](https://scalar.com/) and follows the [OpenAPI 3.1](https://spec.openapis.org/oas/v3.1.0) specification.

You can also access the raw OpenAPI specification at:

http://localhost:8080/openapi.yaml

## License

Apache License 2.0. See [LICENSE](LICENSE).