#!/usr/bin/env bash
# Generates Protobuf code for src/microservices/events from schemas/api/proto.
# Run from the repository root (the codegen container mounts it at /work).
set -euo pipefail

PROTO_DIR=schemas/api/proto

reset_dir() {
  rm -rf "$1"
  mkdir -p "$1"
}

EVENTS=cinemaabyss/events/v1/events.proto

# Go: events-service (Kafka publisher/consumer)
reset_dir src/microservices/events/internal/gen
protoc -I "$PROTO_DIR" \
  --go_out=src/microservices/events --go_opt=module=events-service \
  "$EVENTS"

echo "Generated Go code under src/microservices/events/internal/gen"
