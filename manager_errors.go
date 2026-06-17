package queue

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hibiken/asynq"
)

func mapManagerError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, asynq.ErrTaskNotFound):
		return mapManagerCause(ErrJobNotFound, err)
	case errors.Is(err, asynq.ErrQueueNotFound):
		return mapManagerCause(ErrQueueNotFound, err)
	case errors.Is(err, asynq.ErrQueueNotEmpty):
		return mapManagerCause(ErrQueueNotEmpty, err)
	case isAsynqNotFound(err, "cannot find task"):
		return mapManagerCause(ErrJobNotFound, err)
	case isAsynqNotFound(err, "queue ") && strings.Contains(err.Error(), "does not exist"):
		return mapManagerCause(ErrQueueNotFound, err)
	default:
		return err
	}
}

func mapManagerCause(semantic, cause error) error {
	return fmt.Errorf("%w: %w", semantic, cause)
}

func isAsynqNotFound(err error, text string) bool {
	var debug interface{ DebugString() string }
	return errors.As(err, &debug) &&
		strings.Contains(debug.DebugString(), "NOT_FOUND") &&
		strings.Contains(debug.DebugString(), text)
}

func mapRedisInfoError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	case errors.Is(err, ErrQueueNotFound), errors.Is(err, ErrJobNotFound),
		errors.Is(err, ErrQueueNotEmpty), errors.Is(err, ErrRedisClientNotSupported):
		return err
	default:
		return fmt.Errorf("%w: %w", ErrRedisUnavailable, err)
	}
}
