# dicsord-photo-reaper

Find & Download all files on a discord server and upload them to Google Drive.

This application holds all files in memory to preserve the OS' disk and speed (thanks Gabe for teaching me that one!)

## Usage

### Google Drive Setup

This application requires Google Developer credentials.  

1. Create a project in the [Google Cloud Console](https://cloud.google.com).
1. Enable Google Drive API: Enable the Google Drive API for your project.
1. OAuth 2.0 Credentials: Create OAuth 2.0 credentials (client ID and secret), set up a consent screen, and download the credentials.json file.

### Running the app

First time run, check stdout for 

#### Docker

Container is published as `alex4108/discord_photo_reaper:latest-release`

```
mkdir -p container-volume/
export DISCORD_BOT_TOKEN=""
export DISCORD_GUILD_ID=""
```

Place your client_secret.json from Google in `./container-volume/`

Then `./docker-run.sh`

#### docker-compose

Ensure you replace the values as appropriate before `docker-compose up`

Ensure you place client_secret.json as noted above.

#### Kubernetes

Example manifest at `kube-manifest.yml`

Prometheus scraper endpoint is available on `http://app_server:8889/metrics`

Example prom scrape config:
```
- job_name: discord_photo_reaper
  scrape_interval: 1m
  metrics_path: /metrics
  static_configs:
  - targets:
    - discord-photo-reaper-metrics.discord-photo-reaper.svc.cluster.local:8889
```

## Features

* Stateful runs won't download the same file >1 times
* Rate limit observation / backoff for downloading from discord.

## Development

`make build`

`make test` (no tests?! Look at E2E in this repo)

`make run`

