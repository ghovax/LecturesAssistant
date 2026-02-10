#!/bin/bash
set -e

cd website
npm install
npm run build
cd ..

rm -rf server/internal/api/web/dist
mkdir -p server/internal/api/web/dist
cp -r website/build/* server/internal/api/web/dist/

cd server
mkdir -p dist

GOOS=darwin GOARCH=arm64 go build -o dist/lectures-mac-arm64 ./cmd/server
GOOS=darwin GOARCH=amd64 go build -o dist/lectures-mac-amd64 ./cmd/server
GOOS=linux GOARCH=amd64 go build -o dist/lectures-linux-amd64 ./cmd/server
GOOS=windows GOARCH=amd64 go build -o dist/lectures-windows-amd64.exe ./cmd/server
