package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const sessionDir = "./data/sessions"

// SaveMessage appends a message to the session's JSON file
func SaveMessage(sessionID string, role string, content string) error {
	_ = os.MkdirAll(sessionDir, 0755)
	filePath := filepath.Join(sessionDir, fmt.Sprintf("%s.json", sessionID))

	// Load existing
	session := Session{SessionID: sessionID}
	if data, err := os.ReadFile(filePath); err == nil {
		json.Unmarshal(data, &session)
	}

	// Append new
	session.Messages = append(session.Messages, Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})

	// Save back
	fileData, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, fileData, 0644)
}

// LoadSession retrieves the full conversation for context
func LoadSession(sessionID string) (*Session, error) {
	filePath := filepath.Join(sessionDir, fmt.Sprintf("%s.json", sessionID))
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var session Session
	err = json.Unmarshal(data, &session)
	return &session, err
}
