package main

import (
	"sync"
)

var HTTP_PORT string

// Globals for state tracking
var processedEntities sync.Map
var processedFilePath string
