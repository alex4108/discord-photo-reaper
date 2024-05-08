#!/usr/bin/env bash

ts=$(date +%s)

if [[ "$TAG" == "" ]]; then
    TAG=$(cat .tag)
    if [[ "$TAG" == "" ]]; then
        TAG="alex4108/discord_photo_reaper:latest-release"
    fi
fi

echo "TAG: $TAG"

ct_name=discord-photo-reaper

docker rm $ct_name

set -euo pipefail

# DON'T - Credential Leak!!
# set -x

docker run \
  --name $ct_name \
  -e DISCORD_BOT_TOKEN=$DISCORD_BOT_TOKEN \
  -e DISCORD_GUILD_ID=$DISCORD_GUILD_ID \
  -e LOG_LEVEL=INFO \
  -e GOOGLE_TOKEN_FILE=/host/client_token.json \
  -e GOOGLE_CREDENTIALS_FILE=/host/client_secret.json \
  -e STATE_FILE=/host/discord-photo-reaper.state \
  -v "$(pwd)/container-volume:/host" \
  -p 8888:8888 \
  --restart no \
  "${TAG}"