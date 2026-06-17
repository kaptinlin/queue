package queue

// ListQueues lists all queue names.
func (m *Manager) ListQueues() ([]*QueueInfo, error) {
	queues, err := m.inspector.Queues()
	if err != nil {
		return nil, mapManagerError(err)
	}

	snapshots := make([]*QueueInfo, len(queues))
	for i, queue := range queues {
		qinfo, err := m.inspector.GetQueueInfo(queue)
		if err != nil {
			return nil, mapManagerError(err)
		}
		snapshots[i] = toQueueInfo(qinfo)
	}
	return snapshots, nil
}

// QueueInfo retrieves detailed information about a queue.
func (m *Manager) QueueInfo(queueName string) (*QueueInfo, error) {
	qinfo, err := m.inspector.GetQueueInfo(queueName)
	if err != nil {
		return nil, mapManagerError(err)
	}
	return toQueueInfo(qinfo), nil
}

// ListQueueStats lists statistics for a queue over the past n days.
func (m *Manager) ListQueueStats(queueName string, days int) ([]*QueueDailyStats, error) {
	dstats, err := m.inspector.History(queueName, days)
	if err != nil {
		return nil, mapManagerError(err)
	}

	dailyStats := make([]*QueueDailyStats, len(dstats))
	for i, d := range dstats {
		dailyStats[i] = toQueueDailyStats(d)
	}

	return dailyStats, nil
}

// DeleteQueue deletes a queue by its name.
func (m *Manager) DeleteQueue(queueName string, force bool) error {
	return mapManagerError(m.inspector.DeleteQueue(queueName, force))
}

// PauseQueue pauses a queue by its name.
func (m *Manager) PauseQueue(queueName string) error {
	if err := m.inspector.PauseQueue(queueName); err != nil {
		return mapManagerError(err)
	}
	return nil
}

// ResumeQueue resumes a paused queue by its name.
func (m *Manager) ResumeQueue(queueName string) error {
	if err := m.inspector.UnpauseQueue(queueName); err != nil {
		return mapManagerError(err)
	}
	return nil
}
