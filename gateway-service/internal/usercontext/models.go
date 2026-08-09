package usercontext

import "time"

// UserProfile represents a user's persistent profile and knowledge graph
type UserProfile struct {
	UserID          string          `json:"userID"`
	PhoneNumber     string          `json:"phoneNumber"`
	Role            string          `json:"role"`
	Permissions     []string        `json:"permissions"`
	CoachingGoals   []string        `json:"coachingGoals"`
	CurrentFocus    string          `json:"currentFocus"`
	LastSessionDate time.Time       `json:"lastSessionDate"`
	KnowledgeGraph  *KnowledgeGraph `json:"knowledgeGraph"`
}

// KnowledgeGraph tracks user's goals, patterns, experiments, and learnings
type KnowledgeGraph struct {
	Goals []*Goal `json:"goals"`
}

// Goal represents a coaching goal with associated patterns and experiments
type Goal struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Patterns    []*Pattern     `json:"patterns"`
	Experiments []*Experiment  `json:"experiments"`
	Learnings   []string       `json:"learnings"`
}

// Pattern represents a behavioral pattern or trigger
type Pattern struct {
	Trigger   string `json:"trigger"`
	Behavior  string `json:"behavior"`
	Frequency string `json:"frequency"`
}

// Experiment represents a behavioral experiment or test
type Experiment struct {
	Name    string    `json:"name"`
	Date    time.Time `json:"date"`
	Result  string    `json:"result"` // positive, negative, neutral
	Insight string    `json:"insight"`
}

// ContextLookupRequest is the request to /context/lookup endpoint
type ContextLookupRequest struct {
	PhoneNumber string `json:"phoneNumber" binding:"required"`
}

// ContextLookupResponse is the response from /context/lookup endpoint
type ContextLookupResponse struct {
	UserID            string     `json:"userID"`
	PhoneNumber       string     `json:"phoneNumber"`
	Role              string     `json:"role"`
	Permissions       []string   `json:"permissions"`
	CoachingGoals     []string   `json:"coachingGoals"`
	CurrentFocus      string     `json:"currentFocus"`
	Patterns          []*Pattern `json:"patterns"`
	RecentExperiments []*Experiment `json:"recentExperiments"`
	Learnings         []string   `json:"learnings"`
	LastSessionDate   time.Time  `json:"lastSessionDate"`
}

// ContextUpdateRequest is the request to /context/update endpoint
type ContextUpdateRequest struct {
	UserID           string                 `json:"userID" binding:"required"`
	SessionSummary   map[string]interface{} `json:"sessionSummary"`
	NewLearnings     []string               `json:"newLearnings"`
	NewPattern       *Pattern               `json:"newPattern"`
	NewExperiment    *Experiment            `json:"newExperiment"`
}

// ContextUpdateResponse is the response from /context/update endpoint
type ContextUpdateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	UserID  string `json:"userID"`
}

// HealthCheckResponse is the response from /health endpoint
type HealthCheckResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
