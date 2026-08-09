package usercontext

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// Storage handles JSON file I/O for user profiles
type Storage struct {
	dataDir string
	mu      sync.RWMutex
}

// NewStorage creates a new storage instance with the given data directory
func NewStorage(dataDir string) (*Storage, error) {
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %v", err)
		}
		dataDir = filepath.Join(home, ".adiyan", "users")
	}

	// Ensure directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory %q: %v", dataDir, err)
	}

	log.Printf("[Storage] Initialized with directory: %s", dataDir)
	return &Storage{dataDir: dataDir}, nil
}

// ReadUserFile reads and parses a user profile from disk
func (s *Storage) ReadUserFile(userID string) (*UserProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filePath := s.getUserFilePath(userID)
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("user file not found: %s", userID)
		}
		return nil, fmt.Errorf("failed to read user file %q: %v", filePath, err)
	}

	var user UserProfile
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, fmt.Errorf("failed to parse user JSON: %v", err)
	}

	return &user, nil
}

// WriteUserFile writes a user profile to disk as JSON
func (s *Storage) WriteUserFile(user *UserProfile) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filePath := s.getUserFilePath(user.UserID)

	data, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal user to JSON: %v", err)
	}

	if err := ioutil.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write user file %q: %v", filePath, err)
	}

	log.Printf("[Storage] Wrote user profile: %s", user.UserID)
	return nil
}

// UserFileExists checks if a user profile file exists
func (s *Storage) UserFileExists(userID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filePath := s.getUserFilePath(userID)
	_, err := os.Stat(filePath)
	return err == nil
}

// getUserFilePath returns the full file path for a user profile
func (s *Storage) getUserFilePath(userID string) string {
	return filepath.Join(s.dataDir, fmt.Sprintf("%s.json", userID))
}

// GetDataDir returns the configured data directory
func (s *Storage) GetDataDir() string {
	return s.dataDir
}
