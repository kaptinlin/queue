package queue

import "errors"

// BatchJobError records one failed item from a batch job operation.
type BatchJobError struct {
	JobID string `json:"job_id"`
	Err   error  `json:"-"`
}

// Error implements the error interface.
func (e BatchJobError) Error() string {
	return "job " + e.JobID + ": " + e.Err.Error()
}

// Unwrap returns the underlying cause.
func (e BatchJobError) Unwrap() error {
	return e.Err
}

// BatchJobResult records the successful and failed items from a batch job operation.
type BatchJobResult struct {
	Succeeded []string        `json:"succeeded"`
	Failed    []BatchJobError `json:"failed"`
}

// Err returns all per-job failures joined into one error.
func (r BatchJobResult) Err() error {
	if len(r.Failed) == 0 {
		return nil
	}

	errs := make([]error, len(r.Failed))
	for i, failure := range r.Failed {
		errs[i] = failure
	}
	return errors.Join(errs...)
}

func batchJobOperation(jobIDs []string, operation func(string) error) (BatchJobResult, error) {
	result := BatchJobResult{
		Succeeded: make([]string, 0, len(jobIDs)),
		Failed:    make([]BatchJobError, 0),
	}

	for _, jobID := range jobIDs {
		if err := operation(jobID); err != nil {
			result.Failed = append(result.Failed, BatchJobError{JobID: jobID, Err: err})
			continue
		}
		result.Succeeded = append(result.Succeeded, jobID)
	}

	return result, result.Err()
}
