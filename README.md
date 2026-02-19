# Kanshi Core

Kanshi Core is a simple gRPC server that receives metric batches from Kanshi Agent.

This is v1 — minimal and intentionally basic.

## What It Does

- Starts a gRPC server
- Accepts `IngestBatch` requests
- Logs how many metric points were received
- Returns an acknowledgment

No storage.
No dashboards.
No querying.
Just ingestion.

## Run

```bash
go run cmd/core/main.go
```
The server listens on:
```
:50051
```

## Purpose

This project exists to complete the data pipeline: Agent → [Core](https://github.com/kanshi-dev/core)