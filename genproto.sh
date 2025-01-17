#!/bin/bash

# Create directories if they don't exist
mkdir -p src/apiservice/genproto
mkdir -p src/runnerservice/genproto

# Generate JavaScript code with gRPC
protoc \
    --proto_path=protos \
    --js_out=import_style=commonjs,binary:src/apiservice/genproto \
    --grpc-web_out=import_style=commonjs,mode=grpcwebtext:src/apiservice/genproto \
    protos/main.proto

# Generate Go code with gRPC
protoc \
    --proto_path=protos \
    --go_out=src/runnerservice/genproto \
    --go_opt=paths=source_relative \
    --go-grpc_out=src/runnerservice/genproto \
    --go-grpc_opt=paths=source_relative \
    protos/main.proto

echo "Protocol buffers and gRPC code generated successfully!"

# Set appropriate permissions for generated files
chmod -R 644 src/apiservice/genproto/*
chmod -R 644 src/runnerservice/genproto/*