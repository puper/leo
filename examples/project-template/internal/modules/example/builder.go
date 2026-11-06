package example

import "github.com/puper/leo/v2/engine"

func Builder(container Container) engine.Builder {
	return func() (any, error) { return New(container), nil }
}
