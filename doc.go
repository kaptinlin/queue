// Package queue provides a simple and flexible job queue implementation for Go applications.
// Runtime failures are reported as errors; the package does not expose Must* or other panic-based APIs.
// Integration coverage for client, worker, scheduler, and manager operations requires a live Redis server.
package queue
