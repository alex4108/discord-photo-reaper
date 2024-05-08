#!/usr/bin/env bash

ts=$(date +%s)

image_base="alex4108/discord_photo_reaper"

if [[ "$TAG" == "" ]]; then
    TAG="${image_base}:${ts}"
fi

if [[ "$GITHUB_ACTIONS" != "" ]]; then
    TAG="${image_base}:${TAG}"
fi

set -exuo pipefail

docker build -t ${TAG} .

echo "$TAG" > .tag