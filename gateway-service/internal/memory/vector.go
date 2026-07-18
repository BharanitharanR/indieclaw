package memory

import "context"

// Stubbed functions to satisfy the compiler
func GetEmbedding(input string) ([]float32, error) {
	return []float32{0.0}, nil
}

func Store(ctx context.Context, data string, vec []float32) error {
	return nil
}

func Search(ctx context.Context, vec []float32) (string, error) {
	return "Memory search results", nil
}
