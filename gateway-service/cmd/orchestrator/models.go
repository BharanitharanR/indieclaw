package main

import (
	"time"
)

type Message struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

type Session struct {
	SessionID string    `json:"session_id"`
	Messages  []Message `json:"messages"`
}
