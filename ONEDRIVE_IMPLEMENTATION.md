# OneDrive Storage Implementation

## Overview

This implementation adds support for OneDrive Personal as an alternative cloud storage backend to Google Drive. The application uses a storage provider abstraction that allows users to choose their preferred cloud storage service.

## Architecture

### Storage Interface
The `StorageProvider` interface in `storage.go` defines the contract that all storage implementations must follow:
- `Upload(data *bytes.Buffer, filename string) error` - Uploads a file to cloud storage
- `GetName() string` - Returns the name of the storage provider

### Implementations

#### Google Drive Storage (`gdrive.go`)
- `GoogleDriveStorage` struct wraps the existing Google Drive service
- Uses OAuth 2.0 with Google's API
- Files are uploaded to a folder named "discord-export"
- Maintains backwards compatibility with existing configurations

#### OneDrive Storage (`onedrive.go`)
- `OneDriveStorage` struct uses the OneDrive API (api.onedrive.com)
- Uses OAuth 2.0 with Microsoft Live endpoint for personal Microsoft accounts
- Files are uploaded to a folder named "discord-export"
- Supports personal Microsoft accounts (live.com, outlook.com, hotmail.com)

## Configuration

The storage provider is selected via the `STORAGE_PROVIDER` environment variable:

### Google Drive (Default)
```bash
STORAGE_PROVIDER=gdrive
GOOGLE_CREDENTIALS_FILE=client_secret.json
GOOGLE_TOKEN_FILE=client_token.json
```

### OneDrive
```bash
STORAGE_PROVIDER=onedrive
ONEDRIVE_CLIENT_ID=<your-azure-app-client-id>
ONEDRIVE_TOKEN_FILE=onedrive_token.json
ONEDRIVE_REDIRECT_URL=http://localhost:8888/onedrive  # Optional, this is the default
```

Note: `ONEDRIVE_CLIENT_SECRET` is not required for personal Microsoft accounts when using public client authentication.

## Azure App Registration

To use OneDrive storage, you must register an application in the Azure Portal:

1. Go to [Azure Portal](https://portal.azure.com) > Azure Active Directory > App registrations
2. Click "New registration"
3. Configure:
   - **Name**: Your app name (e.g., "Discord Photo Reaper")
   - **Supported account types**: "Personal Microsoft accounts only"
   - **Redirect URI**: Select "Mobile and desktop applications" and enter `http://localhost:8888/onedrive`
4. After creation, copy the **Application (client) ID** - this is your `ONEDRIVE_CLIENT_ID`

**Important**: The app must be registered as a public client (Mobile and desktop applications), not a web application. Public clients do not require a client secret.

## Authentication Flow

Both storage providers use OAuth 2.0:

1. On first run, if no token file exists, the application starts a local HTTP server on port 8888
2. The user is prompted to visit an authorization URL in their browser
3. After authorization, the user is redirected back to the local server with an authorization code
4. The application exchanges the code for an access token and refresh token
5. The tokens are saved to disk for future use

### OneDrive-Specific Notes
- Uses Microsoft Live endpoint (`login.live.com`) for personal accounts
- The callback endpoint is `/onedrive`
- Requires `onedrive.readwrite` and `offline_access` scopes
- Does not send a client secret (public client authentication)

## API Usage

### OneDrive API Calls

1. **List root folder**: `GET https://api.onedrive.com/v1.0/drive/root/children`
2. **Create folder**: `POST https://api.onedrive.com/v1.0/drive/root/children` with JSON body
3. **Upload file**: `PUT https://api.onedrive.com/v1.0/drive/root:/{folder}/{filename}:/content`

All requests use the OneDrive API v1.0 (`https://api.onedrive.com/v1.0`), not Microsoft Graph API.
This is required when using Microsoft Live authentication with `onedrive.readwrite` scope.

## Code Changes

### Modified Files
- `main.go` - Added `initStorage()` function to initialize the appropriate storage provider
- `discord.go` - Updated to use `StorageProvider` interface instead of direct `drive.Service`
- `fileHandler.go` - Updated `download()` function to accept `StorageProvider`
- `gdrive.go` - Added `GoogleDriveStorage` wrapper implementing `StorageProvider`
- `sample.env` - Added OneDrive configuration options

### New Files
- `storage.go` - Defines the `StorageProvider` interface
- `onedrive.go` - Implements OneDrive storage provider

## Backwards Compatibility

The implementation maintains full backwards compatibility:
- If `STORAGE_PROVIDER` is not set, defaults to Google Drive
- Existing Google Drive configurations continue to work without changes
- All existing metrics and logging continue to function

## Limitations

- Simple upload is limited to files up to 4MB (Discord attachments are typically within this limit)
- Only supports personal Microsoft accounts (not work/school accounts)
- Uses reused Google Drive metrics (TODO: add OneDrive-specific metrics)

## Future Enhancements

Potential improvements:
- Support for work/school accounts (OneDrive for Business) via Microsoft Graph API
- Support for additional storage providers (AWS S3, Dropbox, etc.)
- Configurable folder names per storage provider
- OneDrive-specific metrics
- Support for large file uploads (resumable uploads for files >4MB)
- Multi-storage support (upload to multiple providers simultaneously)
