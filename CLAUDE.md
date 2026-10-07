# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A small Go web service (part of UVA Library's Virgo4 platform) that re-queues a cached record for reindexing. Given an item ID, it looks up the record in a Postgres cache table and pushes it onto an outbound SQS queue as an "update" operation.

## Commands

- `make` / `make darwin` — build `bin/virgo4-cache-reprocess-ws.darwin` (with `-race`)
- `make linux` — static Linux build (used by the Dockerfile)
- `make fmt`, `make vet` — run within `cmd/virgo4-cache-reprocess-ws`
- `make check` — installs and runs `staticcheck` (checks `all,-S1002,-ST1003`) and the `shadow` vet analyzer
- `make dep` — `go get -u`, then `go mod tidy` and `go mod verify`
- Docker image: `docker build -f package/Dockerfile --build-arg BUILD_TAG=<tag> .`

There are no unit tests in this repo. Integration testing happens in CI (`pipeline/testspec.yml`), which clones and runs `uvalib/standard-ws-tester` against the deployed service.

## Architecture

All code is in `package main` under `cmd/virgo4-cache-reprocess-ws/`:

- `config.go` — all config comes from required environment variables (`VIRGO4_CACHE_REPROCESS_WS_*` plus `VIRGO4_SQS_MESSAGE_BUCKET`). A missing or empty variable exits the process.
- `service.go` — `ServiceContext` holds config, the cache proxy, and the SQS client/queue handle (from `github.com/uvalib/virgo4-sqs-sdk/awssqs`). `ReindexHandler` (`PUT /api/reindex/:id`) works like this:
  - cache miss → 404
  - record `source` doesn't match `VIRGO4_CACHE_REPROCESS_WS_DATA_SOURCE` → 422
  - otherwise it sends one SQS message, using `BatchMessagePut` and falling back to `MessagePutRetry` (3 retries)
  - the outbound message always carries `ignore-cache=true` plus record id/type/source and operation=update attributes; the cached payload is the message body
- `cache_proxy.go` — `CacheProxy` interface over Postgres via `ozzo-dbx`. It reads `id, type, source, payload` from the configured table and logs queries slower than 100ms.
- `main.go` — Gin router. Other endpoints: `/version`, `/healthcheck` (DB ping), and Prometheus `/metrics`, which is wired manually to avoid double gzip.
- `version.go` — the version is taken from a `buildtag.<version>` file in the working directory, which the Dockerfile creates from `BUILD_TAG`. Without that file it reports "unknown". The deploy pipeline polls `/version` to confirm a rollout.

## Deployment

AWS CodeBuild specs are in `pipeline/`:
- `buildspec.yml` builds the image, pushes it to ECR, and records the build tag in SSM at `/containers/$CONTAINER_IMAGE/latest`.
- `deployspec.yml` runs Terraform from `github.com/uvalib/terraform-infrastructure` (`virgo4.lib.virginia.edu/ecs-tasks/staging/virgo4-sirsi-cache-reprocess-ws`), then waits for the new version with `wait_for_version.sh`.

Go and Alpine versions are pinned in `package/Dockerfile` and `go.mod`. Routine commits bump these versions and update dependencies.
