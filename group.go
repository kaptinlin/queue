package queue

import (
	"sync"
)

// Group represents a collection of handlers with specific middleware applied.
type Group struct {
	name        string
	worker      *Worker
	middlewares []MiddlewareFunc
	mu          sync.Mutex
}

// Use adds a middleware to the group.
func (g *Group) Use(middlewares ...MiddlewareFunc) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.middlewares = append(g.middlewares, middlewares...)
}

// Register configures and registers a handler for a specific job type within this group.
func (g *Group) Register(jobType string, handle HandlerFunc, opts ...HandlerOption) error {
	if len(g.middlewares) > 0 {
		opts = append([]HandlerOption{WithMiddleware(g.middlewares...)}, opts...)
	}

	return g.worker.Register(jobType, handle, opts...)
}

// RegisterHandler registers a handler for a specific job type within this group.
func (g *Group) RegisterHandler(handler *Handler) error {
	if len(g.middlewares) > 0 {
		handler.Use(g.middlewares...)
	}

	return g.worker.RegisterHandler(handler)
}
