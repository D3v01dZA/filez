#!/bin/sh

set -e

echo "Tagging latest as stable"
docker buildx build --platform linux/amd64,linux/arm64 --push -t d3v01d/filez:stable .
