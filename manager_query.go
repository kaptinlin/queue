package queue

const defaultManagerPageSize = 50

// Page selects a one-based page of manager list results.
type Page struct {
	Size   int
	Number int
}

// JobQuery selects jobs from a queue by state.
type JobQuery struct {
	Queue string
	State JobState
	Group string
	Page  Page
}

func (p Page) normalize() (Page, error) {
	if p.Size == 0 && p.Number == 0 {
		return Page{Size: defaultManagerPageSize, Number: 1}, nil
	}
	if p.Size <= 0 || p.Number <= 0 {
		return Page{}, ErrInvalidPage
	}
	return p, nil
}

func normalizeBatchSize(size int) (int, error) {
	if size <= 0 {
		return 0, ErrInvalidBatchSize
	}
	return size, nil
}
