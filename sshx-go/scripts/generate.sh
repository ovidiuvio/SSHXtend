#!/bin/bash

# Generate protobuf Go code for sshx

set -e

# Make sure we're in the project root
cd "$(dirname "$0")/.."

# Create output directory if it doesn't exist
mkdir -p pkg/proto

# Generate protobuf Go code
protoc --go_out=. --go-grpc_out=. proto/sshx.proto

echo "Protobuf Go code generated successfully"