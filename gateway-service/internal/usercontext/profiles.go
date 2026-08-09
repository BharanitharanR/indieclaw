package usercontext

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// ProfileManager handles user profile loading, creation, and caching
type ProfileManager struct {
	storage *Storage
	cache   map[string]*UserProfile
	cacheMu sync.RWMutex
	cacheTTL time.Duration
	phoneIndex map[string]string // phone -> userID mapping
}

// NewProfileManager creates a new profile manager
func NewProfileManager(storage *Storage) *ProfileManager {
	return &ProfileManager{
		storage:    storage,
		cache:      make(map[string]*UserProfile),
		cacheTTL:   30 * time.Minute,
		phoneIndex: make(map[string]string),
	}
}

// LoadUserByPhone loads a user profile by phone number, creating if needed
func (pm *ProfileManager) LoadUserByPhone(phoneNumber string) (*UserProfile, error) {
	// Check if we have a cached userID for this phone
	pm.cacheMu.RLock()
	if userID, exists := pm.phoneIndex[phoneNumber]; exists {
		if cached, ok := pm.cache[userID]; ok {
			pm.cacheMu.RUnlock()
			log.Printf("[ProfileManager] Cache hit for phone: %s -> %s", phoneNumber, userID)
			return cached, nil
		}
	}
	pm.cacheMu.RUnlock()

	// Try to load from disk
	user, err := pm.storage.ReadUserFile(phoneNumber)
	if err == nil {
		// Found existing user
		pm.cacheUser(user)
		log.Printf("[ProfileManager] Loaded existing user from disk: %s", phoneNumber)
		return user, nil
	}

	// Create new user
	user = pm.createNewUser(phoneNumber)
	if err := pm.storage.WriteUserFile(user); err != nil {
		return nil, fmt.Errorf("failed to save new user: %v", err)
	}
	pm.cacheUser(user)
	log.Printf("[ProfileManager] Created new user: %s -> %s", phoneNumber, user.UserID)
	return user, nil
}

// GetOrCreateUser is an alias for LoadUserByPhone
func (pm *ProfileManager) GetOrCreateUser(phoneNumber string) (*UserProfile, error) {
	return pm.LoadUserByPhone(phoneNumber)
}

// SaveUser saves a user profile to disk and updates cache
func (pm *ProfileManager) SaveUser(user *UserProfile) error {
	if err := pm.storage.WriteUserFile(user); err != nil {
		return err
	}
	pm.cacheUser(user)
	return nil
}

// GetUserByID loads a user by ID from cache or disk
func (pm *ProfileManager) GetUserByID(userID string) (*UserProfile, error) {
	// Check cache first
	pm.cacheMu.RLock()
	if cached, ok := pm.cache[userID]; ok {
		pm.cacheMu.RUnlock()
		return cached, nil
	}
	pm.cacheMu.RUnlock()

	// Load from disk
	user, err := pm.storage.ReadUserFile(userID)
	if err != nil {
		return nil, err
	}
	pm.cacheUser(user)
	return user, nil
}

// cacheUser adds a user to the in-memory cache
func (pm *ProfileManager) cacheUser(user *UserProfile) {
	pm.cacheMu.Lock()
	defer pm.cacheMu.Unlock()
	pm.cache[user.UserID] = user
	pm.phoneIndex[user.PhoneNumber] = user.UserID
}

// createNewUser generates a new user profile with default values
func (pm *ProfileManager) createNewUser(phoneNumber string) *UserProfile {
	userID := fmt.Sprintf("user_%d", time.Now().UnixNano())
	return &UserProfile{
		UserID:          userID,
		PhoneNumber:     phoneNumber,
		Role:            "client",
		Permissions:     []string{"web_search", "coach_framework"},
		CoachingGoals:   []string{},
		CurrentFocus:    "",
		LastSessionDate: time.Now(),
		KnowledgeGraph: &KnowledgeGraph{
			Goals: make([]*Goal, 0),
		},
	}
}
