package queue

type MiddlewareFunc func(HandlerFunc) HandlerFunc
