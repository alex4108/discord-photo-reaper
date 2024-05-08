#!/usr/bin/env bash

ts=$(date +%s)

if [[ "$TAG" == "" ]]; then
    TAG="alex4108/discord_photo_reaper:${ts}"
fi

set -exuo pipefail

docker build -t ${TAG} .

echo "$TAG" > .tag