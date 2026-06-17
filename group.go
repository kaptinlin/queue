package queue

import (
	"slices"
	"sync"
)

// Group represents a collection of handlers with specific middleware applied.
type Group struct {
	worker      *Worker
	middlewares []MiddlewareFunc
	mu          sync.Mutex
}

// Use adds middleware to the group.
func (g *Group) Use(middlewares ...MiddlewareFunc) error {
	if err := g.worker.mutable(); err != nil {
		return err
	}
	for _, middleware := range middlewares {
		if middleware == nil {
			return ErrInvalidMiddleware
		}
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	g.middlewares = append(g.middlewares, middlewares...)
	return nil
}

// Register configures and registers a handler for a specific job type within this group.
func (g *Group) Register(jobType string, handle HandlerFunc, opts ...HandlerOption) error {
	if len(g.middlewares) > 0 {
		opts = slices.Concat([]HandlerOption{WithMiddleware(g.middlewares...)}, opts)
	}

	return g.worker.Register(jobType, handle, opts...)
}

// RegisterHandler registers a handler for a specific job type within this group.
func (g *Group) RegisterHandler(handler *Handler) error {
	if handler != nil && len(g.middlewares) > 0 {
		clone := *handler
		clone.middlewares = slices.Concat(g.middlewares, handler.middlewares)
		clone.composeMiddleware()
		handler = &clone
	}

	return g.worker.RegisterHandler(handler)
}
