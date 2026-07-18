package contracts

import "context"

// Workflow defines what the Gateway expects, without depending on the implementation
type Workflow interface {
	Run(ctx context.Context, input string) (string, error)
}
