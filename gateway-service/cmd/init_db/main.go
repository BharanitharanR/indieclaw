package initdb

import (
	"context"
	"log"

	"github.com/qdrant/go-client/qdrant"
)

func InitializeVectorStore() (*qdrant.Client, error) {
	// Connect using the fast gRPC interface on port 6334
	client, err := qdrant.NewClient(&qdrant.Config{
		Host: "localhost",
		Port: 6334,
	})
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	// Check if the collection already exists to prevent overwrite errors
	exists, err := client.CollectionExists(ctx, "chat_history")
	if err == nil && exists {
		log.Println("Qdrant collection 'chat_history' already initialized.")
		return client, nil
	}

	// Create collection: Nomic embeds are strictly 768 dimensions using Cosine similarity
	err = client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: "chat_history",
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     768,
			Distance: qdrant.Distance_Cosine,
		}),
	})
	if err != nil {
		return nil, err
	}

	log.Println("Successfully created 'chat_history' collection in local Qdrant.")
	return client, nil
}

/*
func main() {
	// 1. Initialize the Qdrant client connection (Default gRPC port is 6334)
	client, err := qdrant.NewClient(&qdrant.Config{
		Host: "localhost",
		Port: 6334,
	})
	if err != nil {
		log.Fatalf("Failed to connect to Qdrant: %v", err)
	}
	defer client.Close()

	// Use a background context with a timeout for database operations
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collectionName := "my_vector_collection"

	// 2. Create a collection (Configuring it for 1536-dim vectors, like OpenAI embeddings)
	err = client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: collectionName,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     1536,
			Distance: qdrant.Distance_Cosine,
		}),
	})
	if err != nil {
		log.Fatalf("Could not create collection: %v", err)
	}

	fmt.Printf("Successfully created collection: %s\n", collectionName)
}
*/
