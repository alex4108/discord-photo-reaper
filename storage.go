package main

import (
	"bytes"
)

// StorageProvider defines the interface for cloud storage providers
type StorageProvider interface {
	// Upload uploads a file to cloud storage
	Upload(data *bytes.Buffer, filename string) error

	// GetName returns the name of the storage provider
	GetName() string
}
