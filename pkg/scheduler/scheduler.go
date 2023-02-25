package scheduler

import (
	"context"
	"time"
)

type Instance struct {
}

func New() (*Instance, error) {
	return nil, nil
}

func (s *Instance) GenerateSchedule(ctx context.Context, past []*Entry, length time.Duration, count int64) ([]*Entry, error) {

	return nil, nil
}
