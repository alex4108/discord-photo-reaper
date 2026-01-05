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
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// GoogleDriveStorage implements StorageProvider for Google Drive
type GoogleDriveStorage struct {
	service *drive.Service
}

// NewGoogleDriveStorage creates a new Google Drive storage provider
func NewGoogleDriveStorage(credentialsFile, tokenFile string) *GoogleDriveStorage {
	service := initGDriveSvc(credentialsFile, tokenFile)
	return &GoogleDriveStorage{service: service}
}

// Upload uploads a file to Google Drive
func (g *GoogleDriveStorage) Upload(data *bytes.Buffer, filename string) error {
	return uploadToGoogleDrive(g.service, data, filename)
}

// GetName returns the storage provider name
func (g *GoogleDriveStorage) GetName() string {
	return "Google Drive"
}

// initGDriveSvc initializes the Google Drive service with OAuth 2.0 credentials
func initGDriveSvc(credentialsFile, tokenFile string) *drive.Service {
	if credentialsFile == "" {
		log.Fatalf("Google credentials file not specified")
	}

	credentials, err := os.ReadFile(credentialsFile)
	if err != nil {
		log.Fatalf("Error reading Google credentials file: %v", err)
	}

	config, err := google.ConfigFromJSON(credentials, drive.DriveScope)

	if os.Getenv("GOOGLE_REDIRECT_URL") != "" {
		config.RedirectURL = os.Getenv("GOOGLE_REDIRECT_URL")
	}

	if err != nil {
		log.Fatalf("Error creating OAuth config: %v", err)
	}

	needFetchToken := false
	var token *oauth2.Token
	if _, err := os.Stat(tokenFile); os.IsNotExist(err) {
		log.Warnf("Token file not found. Fetching new token.")
		needFetchToken = true
	} else {
		token, err = tokenFromFile(tokenFile)
		if err != nil {
			log.Warnf("Error loading OAuth token from file: %v", err)
			needFetchToken = true
		}
	}

	if needFetchToken {
		token, err = fetchInitialToken(config)
		if err != nil {
			log.Fatalf("Failed to fetch initial gdrive token")
		}
		saveTokenToFile(token, tokenFile) // Save the new token to the file
	}

	client := config.Client(context.Background(), token)
	driveService, err := drive.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Error creating Google Drive service: %v", err)
	}

	return driveService
}

// fetchInitialToken starts an HTTP server to receive the OAuth authorization code and exchanges it for an OAuth token
func fetchInitialToken(config *oauth2.Config) (*oauth2.Token, error) {
	authCodeChannel := make(chan string)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		authCode := r.URL.Query().Get("code")
		if authCode == "" {
			http.Error(w, "Authorization code not found", http.StatusBadRequest)
			return
		}
		authCodeChannel <- authCode
		fmt.Fprintf(w, "Authorization successful! You can close this window.")
	})

	HTTP_PORT := "8888"
	if os.Getenv("HTTP_PORT") != "" {
		HTTP_PORT = os.Getenv("HTTP_PORT")
	}

	go func() {
		if err := http.ListenAndServe(":"+HTTP_PORT, nil); err != nil {
			log.Fatalf("Error starting HTTP server: %v", err)
		}
	}()

	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	log.Error("Go to the following link in your browser, authorize the app, and return:")
	log.Errorf("%s", authURL)
	authCode := <-authCodeChannel

	token, err := config.Exchange(context.Background(), authCode)
	if err != nil {
		return nil, fmt.Errorf("error exchanging authorization code for token: %v", err)
	}

	return token, nil
}

// saveTokenToFile saves an OAuth 2.0 token to a file
func saveTokenToFile(token *oauth2.Token, tokenFile string) {
	f, err := os.Create(tokenFile)
	if err != nil {
		log.Fatalf("Error creating token file: %v", err)
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(token); err != nil {
		log.Fatalf("Error encoding token to file: %v", err)
	}
}

// tokenFromFile loads a previously obtained OAuth 2.0 token from a file and validates its validity
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("error opening token file: %v", err)
	}
	defer f.Close()

	token := &oauth2.Token{}
	if err := json.NewDecoder(f).Decode(token); err != nil {
		return nil, fmt.Errorf("error decoding token: %v", err)
	}

	// Check if the token is expired
	if token.Expiry.Before(time.Now()) {
		return nil, fmt.Errorf("OAuth token has expired")
	}

	return token, nil
}

// getOrCreateFolder retrieves the ID of an existing folder by name or creates it if it doesn't exist
func getOrCreateFolder(driveService *drive.Service, folderName string) (string, error) {
	// Search for the folder by name
	query := fmt.Sprintf("name='%s' and mimeType='application/vnd.google-apps.folder'", folderName)
	files, err := driveService.Files.List().Q(query).Do()
	if err != nil {
		return "", fmt.Errorf("error searching for folder %s: %v", folderName, err)
	}

	if len(files.Files) > 0 {
		// If the folder exists, return its ID
		return files.Files[0].Id, nil
	}

	// Create the folder if it doesn't exist
	folderMetadata := &drive.File{
		Name:     folderName,
		MimeType: "application/vnd.google-apps.folder",
	}
	folder, err := driveService.Files.Create(folderMetadata).Do()
	if err != nil {
		return "", fmt.Errorf("error creating folder %s: %v", folderName, err)
	}

	return folder.Id, nil
}

// uploadToGoogleDrive uploads the file from memory to Google Drive in a specified folder
func uploadToGoogleDrive(driveService *drive.Service, data *bytes.Buffer, filename string) error {
	start := time.Now()
	folderName := "discord-export"

	folderID, err := getOrCreateFolder(driveService, folderName)
	if err != nil {
		return fmt.Errorf("error ensuring folder exists: %v", err)
	}

	fileMetadata := &drive.File{
		Name:    filename,
		Parents: []string{folderID}, // Specify the parent folder ID
	}

	uploadedFile, err := driveService.Files.Create(fileMetadata).
		Media(data).
		Do()

	if err != nil {
		return fmt.Errorf("failed to upload %s to Google Drive: %v", filename, err)
	}

	log.Debugf("File uploaded to Google Drive in folder %s with ID: %s", folderName, uploadedFile.Id)
	googleDriveUploadDuration.WithLabelValues().Observe(float64(time.Since(start).Seconds()))

	return nil
}
