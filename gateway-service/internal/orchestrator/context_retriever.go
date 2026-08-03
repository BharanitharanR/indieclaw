package orchestrator

import (
	"context"
	"log"

	"github.com/qdrant/go-client/qdrant"
)

type QdrantContextRetriever struct {
	client *qdrant.Client
	embeddingModel string
}

func NewQdrantContextRetriever(client *qdrant.Client, embeddingModel string) *QdrantContextRetriever {
	return &QdrantContextRetriever{
		client: client,
		embeddingModel: embeddingModel,
	}
}

func (r *QdrantContextRetriever) Retrieve(ctx context.Context, input string, sessionID string, limit int) ([]RetrievedContext, error) {
	if r.client == nil {
		log.Printf("⚠️  Qdrant client not available, skipping semantic search")
		return []RetrievedContext{}, nil
	}

	queryVector, err := EmbedQuery(ctx, r.embeddingModel, input)
	if err != nil {
		log.Printf("❌ Failed to embed query: %v", err)
		return nil, err
	}

	log.Printf("🔍 Searching Qdrant for semantic matches (limit: %d)", limit)

	limitVal := uint64(limit)
	searchResult, err := r.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: "chat_history",
		Query:          qdrant.NewQuery(queryVector...),
		Limit:          &limitVal,
		WithPayload:    qdrant.NewWithPayload(true),
	})

	if err != nil {
		log.Printf("❌ Qdrant query failed: %v", err)
		return nil, err
	}

	var contexts []RetrievedContext
	for i, point := range searchResult {
		payload := point.Payload

		content := ""
		if payloadValue, exists := payload["content"]; exists {
			if strValue, ok := payloadValue.GetKind().(*qdrant.Value_StringValue); ok {
				content = strValue.StringValue
			}
		}

		retrievedSessionID := ""
		if payloadValue, exists := payload["session_id"]; exists {
			if strValue, ok := payloadValue.GetKind().(*qdrant.Value_StringValue); ok {
				retrievedSessionID = strValue.StringValue
			}
		}

		context := RetrievedContext{
			Content:    content,
			Similarity: float64(point.Score),
			Source:     "qdrant",
			SessionID:  retrievedSessionID,
		}

		contexts = append(contexts, context)
		log.Printf("  [%d] Score: %.2f | SessionID: %s | Content length: %d", i+1, point.Score, retrievedSessionID, len(content))
	}

	log.Printf("✅ Retrieved %d semantic contexts from Qdrant", len(contexts))
	return contexts, nil
}
