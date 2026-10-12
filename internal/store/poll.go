package store

import (
	"context"
	"time"
)

// Poll calls Reload every `every` until ctx is done. onChange gets the
// relative paths of files that changed (never an empty slice); onError gets
// every error Reload returns, including *ParseError.
// ponytail: mtime poll, swap for fsnotify if an engagement ever has hundreds of hosts.
func (s *Store) Poll(
	ctx context.Context,
	every time.Duration,
	onChange func(changed []string),
	onError func(err error),
) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			changed, err := s.Reload()
			if err != nil {
				onError(err)
			}
			if len(changed) > 0 {
				onChange(changed)
			}
		}
	}
}
