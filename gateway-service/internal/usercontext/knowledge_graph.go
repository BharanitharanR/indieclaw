package usercontext

import (
	"fmt"
	"log"
	"time"
)

// KnowledgeGraphManager handles operations on the user's knowledge graph
type KnowledgeGraphManager struct {
	profileManager *ProfileManager
}

// NewKnowledgeGraphManager creates a new knowledge graph manager
func NewKnowledgeGraphManager(pm *ProfileManager) *KnowledgeGraphManager {
	return &KnowledgeGraphManager{profileManager: pm}
}

// AddGoal adds a new goal to a user's knowledge graph
func (kg *KnowledgeGraphManager) AddGoal(user *UserProfile, goal *Goal) error {
	if goal.ID == "" {
		goal.ID = fmt.Sprintf("goal_%d", time.Now().UnixNano())
	}
	if goal.Patterns == nil {
		goal.Patterns = make([]*Pattern, 0)
	}
	if goal.Experiments == nil {
		goal.Experiments = make([]*Experiment, 0)
	}
	if goal.Learnings == nil {
		goal.Learnings = make([]string, 0)
	}

	user.KnowledgeGraph.Goals = append(user.KnowledgeGraph.Goals, goal)
	log.Printf("[KnowledgeGraph] Added goal %q to user %s", goal.Name, user.UserID)
	return kg.profileManager.SaveUser(user)
}

// AddPattern adds a pattern to a goal
func (kg *KnowledgeGraphManager) AddPattern(user *UserProfile, goalID string, pattern *Pattern) error {
	goal := kg.findGoal(user, goalID)
	if goal == nil {
		return fmt.Errorf("goal not found: %s", goalID)
	}

	goal.Patterns = append(goal.Patterns, pattern)
	log.Printf("[KnowledgeGraph] Added pattern %q to goal %s", pattern.Trigger, goalID)
	return kg.profileManager.SaveUser(user)
}

// AddExperiment adds an experiment to a goal
func (kg *KnowledgeGraphManager) AddExperiment(user *UserProfile, goalID string, exp *Experiment) error {
	goal := kg.findGoal(user, goalID)
	if goal == nil {
		return fmt.Errorf("goal not found: %s", goalID)
	}

	if exp.Date.IsZero() {
		exp.Date = time.Now()
	}

	goal.Experiments = append(goal.Experiments, exp)
	log.Printf("[KnowledgeGraph] Added experiment %q to goal %s", exp.Name, goalID)
	return kg.profileManager.SaveUser(user)
}

// AddLearning adds a learning to a goal
func (kg *KnowledgeGraphManager) AddLearning(user *UserProfile, goalID string, learning string) error {
	goal := kg.findGoal(user, goalID)
	if goal == nil {
		return fmt.Errorf("goal not found: %s", goalID)
	}

	goal.Learnings = append(goal.Learnings, learning)
	log.Printf("[KnowledgeGraph] Added learning to goal %s: %q", goalID, learning)
	return kg.profileManager.SaveUser(user)
}

// GetGoalPatterns retrieves all patterns for a goal
func (kg *KnowledgeGraphManager) GetGoalPatterns(user *UserProfile, goalID string) []*Pattern {
	goal := kg.findGoal(user, goalID)
	if goal == nil {
		return make([]*Pattern, 0)
	}
	return goal.Patterns
}

// GetGoalExperiments retrieves all experiments for a goal
func (kg *KnowledgeGraphManager) GetGoalExperiments(user *UserProfile, goalID string) []*Experiment {
	goal := kg.findGoal(user, goalID)
	if goal == nil {
		return make([]*Experiment, 0)
	}
	return goal.Experiments
}

// GetGoalLearnings retrieves all learnings for a goal
func (kg *KnowledgeGraphManager) GetGoalLearnings(user *UserProfile, goalID string) []string {
	goal := kg.findGoal(user, goalID)
	if goal == nil {
		return make([]string, 0)
	}
	return goal.Learnings
}

// GetAllPatterns retrieves all patterns across all goals
func (kg *KnowledgeGraphManager) GetAllPatterns(user *UserProfile) []*Pattern {
	patterns := make([]*Pattern, 0)
	for _, goal := range user.KnowledgeGraph.Goals {
		patterns = append(patterns, goal.Patterns...)
	}
	return patterns
}

// GetAllLearnings retrieves all learnings across all goals
func (kg *KnowledgeGraphManager) GetAllLearnings(user *UserProfile) []string {
	learnings := make([]string, 0)
	for _, goal := range user.KnowledgeGraph.Goals {
		learnings = append(learnings, goal.Learnings...)
	}
	return learnings
}

// GetRecentExperiments retrieves the most recent experiments (up to n)
func (kg *KnowledgeGraphManager) GetRecentExperiments(user *UserProfile, n int) []*Experiment {
	experiments := make([]*Experiment, 0)

	// Collect all experiments
	for _, goal := range user.KnowledgeGraph.Goals {
		experiments = append(experiments, goal.Experiments...)
	}

	// Sort by date (most recent first) - simple bubble sort for now
	for i := 0; i < len(experiments)-1; i++ {
		for j := 0; j < len(experiments)-i-1; j++ {
			if experiments[j].Date.Before(experiments[j+1].Date) {
				experiments[j], experiments[j+1] = experiments[j+1], experiments[j]
			}
		}
	}

	// Return top n
	if n > len(experiments) {
		n = len(experiments)
	}
	return experiments[:n]
}

// findGoal finds a goal by ID
func (kg *KnowledgeGraphManager) findGoal(user *UserProfile, goalID string) *Goal {
	for _, goal := range user.KnowledgeGraph.Goals {
		if goal.ID == goalID {
			return goal
		}
	}
	return nil
}
