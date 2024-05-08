#!/usr/bin/env bash

set -exo pipefail

if [[ "$TAG" == "" ]]; then
    TAG=$(cat .tag)
    if [[ "$TAG" == "" ]]; then
        echo "No TAG"
        exit 1
    fi
fi

set -u

latest_tag="$(echo $TAG | cut -d: -f1):latest-release"

docker tag ${TAG} $latest_tag
docker push ${TAG}
docker push $latest_tag
