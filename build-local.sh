#!/bin/bash

# Define version
VERSION="2.0.15"

# Define dockerfiles and tags
declare -A dockerfiles=( ["janet"]="./cmd/janet/Dockerfile"  )
declare -A tags=( ["janet"]="troyxmccall/janet" )

# Build docker images
for id in "${!dockerfiles[@]}"; do
  docker build -t ${tags[$id]}:${VERSION} -t ${tags[$id]}:latest -f ${dockerfiles[$id]} .
  if [ "$id" == "janet-webui" ]; then
    docker tag ${tags[$id]}:${VERSION} ${tags[$id]}:${VERSION}-webui
    docker tag ${tags[$id]}:latest ${tags[$id]}:latest-webui
  fi
done
