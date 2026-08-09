package usercontext

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Server handles HTTP requests for the User Context Service
type Server struct {
	profileManager    *ProfileManager
	knowledgeManager  *KnowledgeGraphManager
	port              string
	server            *http.Server
}

// NewServer creates a new User Context Service server
func NewServer(pm *ProfileManager, km *KnowledgeGraphManager, port string) *Server {
	if port == "" {
		port = ":8001"
	}

	s := &Server{
		profileManager:   pm,
		knowledgeManager: km,
		port:             port,
	}

	// Setup routes
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/context/lookup", s.handleContextLookup)
	mux.HandleFunc("/context/update", s.handleContextUpdate)
	mux.HandleFunc("/users/", s.handleGetUser)

	s.server = &http.Server{
		Addr:         s.port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("[Server] Starting User Context Service on %s", s.port)
	return s.server.ListenAndServe()
}

// Stop gracefully stops the server
func (s *Server) Stop() error {
	log.Printf("[Server] Stopping User Context Service")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}

// handleHealth handles GET /health requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(HealthCheckResponse{
		Status:  "ok",
		Message: "User Context Service is running",
	})
}

// handleContextLookup handles POST /context/lookup requests
func (s *Server) handleContextLookup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	phoneNumber := r.URL.Query().Get("phoneNumber")
	if phoneNumber == "" {
		http.Error(w, "Missing phoneNumber parameter", http.StatusBadRequest)
		return
	}

	log.Printf("[Server] Context lookup for phoneNumber: %s", phoneNumber)

	user, err := s.profileManager.LoadUserByPhone(phoneNumber)
	if err != nil {
		log.Printf("[Server] Error loading user: %v", err)
		http.Error(w, "Failed to load user context", http.StatusInternalServerError)
		return
	}

	// Build response
	resp := ContextLookupResponse{
		UserID:          user.UserID,
		PhoneNumber:     user.PhoneNumber,
		Role:            user.Role,
		Permissions:     user.Permissions,
		CoachingGoals:   user.CoachingGoals,
		CurrentFocus:    user.CurrentFocus,
		Patterns:        s.knowledgeManager.GetAllPatterns(user),
		RecentExperiments: s.knowledgeManager.GetRecentExperiments(user, 5),
		Learnings:       s.knowledgeManager.GetAllLearnings(user),
		LastSessionDate: user.LastSessionDate,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
	log.Printf("[Server] Returned context for user: %s", user.UserID)
}

// handleContextUpdate handles POST /context/update requests
func (s *Server) handleContextUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ContextUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" {
		http.Error(w, "Missing userID", http.StatusBadRequest)
		return
	}

	log.Printf("[Server] Context update for user: %s", req.UserID)

	user, err := s.profileManager.GetUserByID(req.UserID)
	if err != nil {
		log.Printf("[Server] User not found: %s", req.UserID)
		http.Error(w, fmt.Sprintf("User not found: %s", req.UserID), http.StatusNotFound)
		return
	}

	// Update last session date
	user.LastSessionDate = time.Now()

	// Add new learnings if provided
	if len(req.NewLearnings) > 0 && len(user.KnowledgeGraph.Goals) > 0 {
		// Add to the first goal for now (could be enhanced to specify which goal)
		firstGoal := user.KnowledgeGraph.Goals[0]
		for _, learning := range req.NewLearnings {
			firstGoal.Learnings = append(firstGoal.Learnings, learning)
		}
		log.Printf("[Server] Added %d learnings to goal %s", len(req.NewLearnings), firstGoal.ID)
	}

	// Add new pattern if provided
	if req.NewPattern != nil && len(user.KnowledgeGraph.Goals) > 0 {
		firstGoal := user.KnowledgeGraph.Goals[0]
		firstGoal.Patterns = append(firstGoal.Patterns, req.NewPattern)
		log.Printf("[Server] Added pattern to goal %s", firstGoal.ID)
	}

	// Add new experiment if provided
	if req.NewExperiment != nil && len(user.KnowledgeGraph.Goals) > 0 {
		if req.NewExperiment.Date.IsZero() {
			req.NewExperiment.Date = time.Now()
		}
		firstGoal := user.KnowledgeGraph.Goals[0]
		firstGoal.Experiments = append(firstGoal.Experiments, req.NewExperiment)
		log.Printf("[Server] Added experiment to goal %s", firstGoal.ID)
	}

	// Save updated user
	if err := s.profileManager.SaveUser(user); err != nil {
		log.Printf("[Server] Error saving user: %v", err)
		http.Error(w, "Failed to save user context", http.StatusInternalServerError)
		return
	}

	resp := ContextUpdateResponse{
		Success: true,
		Message: "User context updated",
		UserID:  req.UserID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
	log.Printf("[Server] Saved updated context for user: %s", req.UserID)
}

// handleGetUser handles GET /users/{userID} debug endpoint
func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Path[len("/users/"):]
	if userID == "" {
		http.Error(w, "Missing userID", http.StatusBadRequest)
		return
	}

	user, err := s.profileManager.GetUserByID(userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
