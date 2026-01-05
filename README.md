# dicsord-photo-reaper

Find & Download all files on a discord server and upload them to cloud storage (Google Drive or OneDrive).

This application holds all files in memory to preserve the OS' disk and speed (thanks Gabe for teaching me that one!)

## Usage

### Storage Provider Configuration

This application now supports both Google Drive and OneDrive as storage backends. Configure your preferred storage provider using the `STORAGE_PROVIDER` environment variable.

#### Google Drive Setup

1. Create a project in the [Google Cloud Console](https://cloud.google.com).
2. Enable Google Drive API: Enable the Google Drive API for your project.
3. OAuth 2.0 Credentials: Create OAuth 2.0 credentials (client ID and secret), set up a consent screen, and download the credentials.json file.
4. Set environment variables:
   ```
   STORAGE_PROVIDER=gdrive
   GOOGLE_CREDENTIALS_FILE=client_secret.json
   GOOGLE_TOKEN_FILE=client_token.json
   ```

#### OneDrive Setup (Personal Microsoft Accounts)

1. Register an application in the [Azure Portal](https://portal.azure.com/#blade/Microsoft_AAD_RegisteredApps/ApplicationsListBlade).
2. Navigate to "Azure Active Directory" > "App registrations" > "New registration".
3. Configure your app:
   - **Name**: Choose a name (e.g., "Discord Photo Reaper")
   - **Supported account types**: Select "Personal Microsoft accounts only"
   - **Redirect URI**: Select "Mobile and desktop applications" and enter `http://localhost:8888/onedrive`
4. After creation, copy the **Application (client) ID** from the overview page.
5. Set environment variables:
   ```
   STORAGE_PROVIDER=onedrive
   ONEDRIVE_CLIENT_ID=<your-client-id>
   ONEDRIVE_TOKEN_FILE=onedrive_token.json
   ```

**Note**: No client secret is required for personal Microsoft accounts when using public client authentication.

On first run, the application will prompt you to authorize access via a browser window for both storage providers.

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

