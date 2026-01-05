package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

// MicrosoftLiveEndpoint is the OAuth2 endpoint for personal Microsoft accounts.
// This uses the Microsoft Account (MSA) authentication endpoint, which is required
// for OneDrive Personal accounts (live.com, outlook.com, hotmail.com).
// Note: This is different from Azure AD endpoints used for work/school accounts.
var MicrosoftLiveEndpoint = oauth2.Endpoint{
	AuthURL:   "https://login.live.com/oauth20_authorize.srf",
	TokenURL:  "https://login.live.com/oauth20_token.srf",
	AuthStyle: oauth2.AuthStyleInParams, // Required: send credentials in POST body, not HTTP Basic Auth
}

// OneDriveStorage implements StorageProvider for OneDrive Personal accounts.
// It uses the OneDrive API (api.onedrive.com) rather than Microsoft Graph API.
type OneDriveStorage struct {
	client *http.Client
	config *oauth2.Config
}

// NewOneDriveStorage creates a new OneDrive storage provider for personal Microsoft accounts.
// The clientSecret parameter is accepted for API compatibility but is not used, as personal
// Microsoft accounts use public client authentication (no client secret required).
//
// Required Azure App Registration settings:
//   - Supported account types: "Personal Microsoft accounts only"
//   - Platform: "Mobile and desktop applications"
//   - Redirect URI: http://localhost:8888/onedrive (or custom via ONEDRIVE_REDIRECT_URL)
func NewOneDriveStorage(clientID, clientSecret, tokenFile string) *OneDriveStorage {
	// For personal Microsoft accounts (public client apps), we don't send a client secret.
	// The Azure app must be registered as a public client (Mobile and desktop applications).
	config := &oauth2.Config{
		ClientID: clientID,
		Endpoint: MicrosoftLiveEndpoint,
		Scopes:   []string{"onedrive.readwrite", "offline_access"},
	}

	if os.Getenv("ONEDRIVE_REDIRECT_URL") != "" {
		config.RedirectURL = os.Getenv("ONEDRIVE_REDIRECT_URL")
	} else {
		config.RedirectURL = "http://localhost:8888/onedrive"
	}

	needFetchToken := false
	var token *oauth2.Token
	var err error

	if _, err := os.Stat(tokenFile); os.IsNotExist(err) {
		log.Warnf("OneDrive token file not found. Fetching new token.")
		needFetchToken = true
	} else {
		token, err = tokenFromFile(tokenFile)
		if err != nil {
			log.Warnf("Error loading OneDrive OAuth token from file: %v", err)
			needFetchToken = true
		}
	}

	if needFetchToken {
		token, err = fetchOneDriveToken(config)
		if err != nil {
			log.Fatalf("Failed to fetch initial OneDrive token: %v", err)
		}
		saveTokenToFile(token, tokenFile)
	}

	client := config.Client(context.Background(), token)
	return &OneDriveStorage{
		client: client,
		config: config,
	}
}

// Upload uploads a file to OneDrive
func (o *OneDriveStorage) Upload(data *bytes.Buffer, filename string) error {
	return uploadToOneDrive(o.client, data, filename)
}

// GetName returns the storage provider name
func (o *OneDriveStorage) GetName() string {
	return "OneDrive"
}

// fetchOneDriveToken starts a local HTTP server to receive the OAuth authorization code
// via the redirect URI callback. The user must visit the authorization URL in their browser
// and grant permission to the application.
func fetchOneDriveToken(config *oauth2.Config) (*oauth2.Token, error) {
	authCodeChannel := make(chan string)
	errorChannel := make(chan error)

	mux := http.NewServeMux()
	mux.HandleFunc("/onedrive", func(w http.ResponseWriter, r *http.Request) {
		authCode := r.URL.Query().Get("code")
		if authCode == "" {
			http.Error(w, "Authorization code not found", http.StatusBadRequest)
			errorChannel <- fmt.Errorf("authorization code not found")
			return
		}
		authCodeChannel <- authCode
		fmt.Fprintf(w, "OneDrive authorization successful! You can close this window.")
	})

	HTTP_PORT := "8888"
	if os.Getenv("HTTP_PORT") != "" {
		HTTP_PORT = os.Getenv("HTTP_PORT")
	}

	server := &http.Server{
		Addr:    ":" + HTTP_PORT,
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorf("Error starting HTTP server: %v", err)
		}
	}()

	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	log.Error("Go to the following link in your browser, authorize the app, and return:")
	log.Errorf("%s", authURL)

	select {
	case authCode := <-authCodeChannel:
		log.Debugf("Received authorization code, exchanging for token...")
		// Shutdown the server after receiving the code
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			server.Shutdown(ctx)
		}()

		token, err := config.Exchange(context.Background(), authCode)
		if err != nil {
			return nil, fmt.Errorf("error exchanging authorization code for token: %v", err)
		}
		log.Infof("Successfully obtained OneDrive OAuth token")
		return token, nil
	case err := <-errorChannel:
		return nil, err
	}
}

// getOrCreateOneDriveFolder retrieves the ID of an existing folder by name or creates it if it doesn't exist.
// Uses the OneDrive API (api.onedrive.com) which is required for personal Microsoft accounts
// when authenticating via the Microsoft Live endpoint with onedrive.readwrite scope.
func getOrCreateOneDriveFolder(client *http.Client, folderName string) (string, error) {
	listURL := "https://api.onedrive.com/v1.0/drive/root/children"

	resp, err := client.Get(listURL)
	if err != nil {
		return "", fmt.Errorf("error listing root folder: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error listing root folder, status: %d", resp.StatusCode)
	}

	var result struct {
		Value []struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Folder *struct {
				ChildCount int `json:"childCount"`
			} `json:"folder,omitempty"`
		} `json:"value"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("error decoding list response: %v", err)
	}

	// Check if folder exists
	for _, item := range result.Value {
		if item.Name == folderName && item.Folder != nil {
			return item.ID, nil
		}
	}

	// Create the folder if it doesn't exist
	createURL := "https://api.onedrive.com/v1.0/drive/root/children"
	folderData := map[string]interface{}{
		"name":                   folderName,
		"folder":                 map[string]interface{}{},
		"@name.conflictBehavior": "rename",
	}

	jsonData, err := json.Marshal(folderData)
	if err != nil {
		return "", fmt.Errorf("error marshaling folder data: %v", err)
	}

	req, err := http.NewRequest("POST", createURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err = client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error creating folder: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("error creating folder, status: %d", resp.StatusCode)
	}

	var folder struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&folder); err != nil {
		return "", fmt.Errorf("error decoding folder response: %v", err)
	}

	return folder.ID, nil
}

// uploadToOneDrive uploads a file from memory to the "discord-export" folder in OneDrive.
// Uses simple upload (PUT request) which supports files up to 4MB. For larger files,
// OneDrive's resumable upload API should be used instead.
func uploadToOneDrive(client *http.Client, data *bytes.Buffer, filename string) error {
	start := time.Now()
	folderName := "discord-export"

	// Ensure the target folder exists (creates it if needed)
	_, err := getOrCreateOneDriveFolder(client, folderName)
	if err != nil {
		return fmt.Errorf("error ensuring OneDrive folder exists: %v", err)
	}

	// Upload file using path-based addressing
	uploadURL := fmt.Sprintf("https://api.onedrive.com/v1.0/drive/root:/%s/%s:/content", folderName, filename)

	req, err := http.NewRequest("PUT", uploadURL, data)
	if err != nil {
		return fmt.Errorf("error creating upload request: %v", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload %s to OneDrive: %v", filename, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to upload %s to OneDrive, status: %d", filename, resp.StatusCode)
	}

	var uploadResult struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&uploadResult); err != nil {
		log.Warnf("Could not decode upload response, but upload may have succeeded")
	}

	log.Debugf("File uploaded to OneDrive in folder %s with ID: %s", folderName, uploadResult.ID)

	// Record upload duration metric (reuses Google Drive metric for now)
	// TODO: Add OneDrive-specific metrics
	googleDriveUploadDuration.WithLabelValues().Observe(float64(time.Since(start).Seconds()))

	return nil
}
