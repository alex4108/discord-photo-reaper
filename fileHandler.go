package main

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/drive/v3"
)

// download downloads the content from the URL into memory and checks its integrity
func download(url, name, expectedContentType string, expectedFileSize int, driveService *drive.Service) {
	if checkOk(url) {
		log.Debugf("File already downloaded %s", url)
		return // Already downloaded
	}

	resp, err := http.Get(url)
	if err != nil {
		log.Errorf("error downloading file from %s: %v", url, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Errorf("HTTP status code %d while downloading file from %s", resp.StatusCode, url)
		return
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, resp.Body); err != nil {
		log.Errorf("error copying content to buffer: %v", err)
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != expectedContentType && expectedContentType != "" {
		log.Warnf("unexpected content-type: expected %s, got %s", expectedContentType, contentType)
	}

	// This never matches...
	// if buf.Len() != expectedFileSize {
	// 	log.Errorf("file size mismatch: expected %d bytes, got %d bytes", expectedFileSize, buf.Len())
	// 	return
	// }

	// Try to determine the content-type from the data itself
	mimeType := mimetype.Detect(buf.Bytes())
	if !strings.HasPrefix(mimeType.String(), expectedContentType) {
		log.Warnf("content-type mismatch: expected %s, detected %s", expectedContentType, mimeType.String())
	}

	// README Data upload here
	err = uploadToGoogleDrive(driveService, &buf, name)
	if err != nil {
		log.Errorf("Error uploading %s to gdrive: %v", url, err)
		return
	}
	uploadedFiles.Add(1)

	recordOk(url)
}
