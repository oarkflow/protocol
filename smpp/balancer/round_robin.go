package balancer

import (
	"sync/atomic"

	"github.com/oarkflow/errors"
)

var (
	// ErrNoAvailableItem no item is available
	ErrNoAvailableItem = errors.New("no item is available")
)

type RoundRobin struct {
	index uint32
}

func (r *RoundRobin) Pick(ids []string) (string, error) {
	if len(ids) == 0 {
		return "", ErrNoAvailableItem
	}

	index := atomic.AddUint32(&r.index, 1) % uint32(len(ids))
	return ids[index], nil
}
