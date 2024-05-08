package main

import (
	"sync"

	"google.golang.org/api/drive/v3"
)

var driveService *drive.Service

var HTTP_PORT string

// Globals for state tracking
var processedUrls sync.Map
var processedFilePath string
