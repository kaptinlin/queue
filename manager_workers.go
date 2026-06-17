package queue

// ListWorkers retrieves information about all Asynq servers (workers) and the jobs they are currently processing.
func (m *Manager) ListWorkers() ([]*WorkerInfo, error) {
	servers, err := m.inspector.Servers()
	if err != nil {
		return nil, mapManagerError(err)
	}

	workers := make([]*WorkerInfo, len(servers))
	for i, server := range servers {
		workers[i] = toWorkerInfo(server)
	}
	return workers, nil
}

// WorkerInfo retrieves detailed information about a single worker using its ID.
func (m *Manager) WorkerInfo(workerID string) (*WorkerInfo, error) {
	servers, err := m.inspector.Servers()
	if err != nil {
		return nil, mapManagerError(err)
	}

	for _, server := range servers {
		if server.ID == workerID {
			return toWorkerInfo(server), nil
		}
	}

	return nil, ErrWorkerNotFound
}
