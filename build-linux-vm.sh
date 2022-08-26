#!/bin/bash
echo "Building for linux/amd64 ..."
env GOOS=linux GOARCH=amd64 go build
echo "Done"