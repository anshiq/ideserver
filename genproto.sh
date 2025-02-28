#!/bin/bash

set -e  # Exit immediately if any command fails

# Function to handle errors
die() {
    echo "Error: $1" >&2
    exit 1
}

# Create directories if they don't exist
mkdir -p src/apiservice/genproto || die "Failed to create directory src/apiservice/genproto"
mkdir -p src/runnerservice/genproto || die "Failed to create directory src/runnerservice/genproto"

# Copy proto file
cp ./protos/main.proto ./src/apiservice/genproto || die "Failed to copy main.proto"

# Generate Go code with gRPC
protoc \
    --proto_path=protos \
    --go_out=src/runnerservice/genproto \
    --go_opt=paths=source_relative \
    --go-grpc_out=src/runnerservice/genproto \
    --go-grpc_opt=paths=source_relative \
    protos/main.proto || die "protoc Go generation failed"

# Set appropriate permissions for generated files
chmod -R 644 src/apiservice/genproto/* || die "Failed to set permissions for src/apiservice/genproto"
chmod -R 644 src/runnerservice/genproto/* || die "Failed to set permissions for src/runnerservice/genproto"

echo "Protocol buffers and gRPC code generated successfully!"
