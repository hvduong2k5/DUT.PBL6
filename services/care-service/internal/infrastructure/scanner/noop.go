package scanner

import "context"

// Noop is allowed only outside production. Config validation prevents production startup with this scanner.
type Noop struct{}

func (Noop) Scan(context.Context, string) error {
	return nil
}
