# Security policy

## Supported versions

The latest stable minor release receives security fixes.

## Deployment boundary

OTLP/gRPC uses the same required `x-api-key` as Agent ingestion. Keep port `50051` private or behind TLS, rotate the shared key if exposed, and redact secrets before telemetry reaches the Collector. Core rejects oversized, malformed, non-finite, future, and retention-expired telemetry.

## Report a vulnerability

Use GitHub's private vulnerability reporting for this repository. Do not open a public issue. Include affected versions, reproduction steps, impact, and any suggested mitigation. The maintainers will acknowledge the report and coordinate disclosure through the private advisory.
