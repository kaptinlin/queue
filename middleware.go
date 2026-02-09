package queue

// MiddlewareFunc defines a function that wraps a [HandlerFunc] to add
// pre-processing or post-processing logic. Middleware functions are
// composed in the order they are added, with the first middleware being
// the outermost wrapper.
type MiddlewareFunc func(HandlerFunc) HandlerFunc
