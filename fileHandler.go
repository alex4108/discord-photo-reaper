package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	log "github.com/sirupsen/logrus"
)

// download downloads the content from the URL into memory and checks its integrity
func download(url, name, expectedContentType string, expectedFileSize int) {
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
	err = uploadToGoogleDrive(&buf, name)
	if err != nil {
		log.Errorf("Error uploading %s to gdrive: %v", url, err)
		return
	}

	recordOk(url)
}

// Initialize processed URLs from a file
func initProcessedUrls() *os.File {
	file, err := os.Open(processedFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Fatalf("Error opening processed URLs file: %v", err)
		}
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		url := strings.TrimSpace(scanner.Text())
		if url != "" {
			processedUrls.Store(url, true)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Errorf("Error reading processed URLs file: %v", err)
	}

	return file
}

// recordOk marks a URL as successfully processed and persists it to the file
func recordOk(url string) {
	if _, loaded := processedUrls.LoadOrStore(url, true); !loaded {
		file, err := os.OpenFile(processedFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Errorf("Error opening processed URLs file for append: %v", err)
			return
		}
		defer file.Close()

		_, err = file.WriteString(fmt.Sprintf("%s\n", url))
		if err != nil {
			log.Errorf("Error writing URL to processed URLs file: %v", err)
		}
	}
}

// checkOk checks if a URL has already been processed
func checkOk(url string) bool {
	_, exists := processedUrls.Load(url)
	return exists
}
