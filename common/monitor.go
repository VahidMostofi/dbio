// Monitor watches a file every `interval` duration and reports if it is changed or not
// in comparision to the last version.
package main

import (
	"context"
	"crypto/md5"
	"io/ioutil"
	"os"
	"time"
)

// Monitor watches specifies which file should be watch.
type Monitor struct {
	lastHash [16]byte
	first    bool
	path     string
}

// NewMonitor constructs and returns a new *Monitor instance
func NewMonitor(filePath string) *Monitor {
	m := &Monitor{
		first: true,
		path:  filePath,
	}

	return m
}

// watch periodically checks the source that is defined at filePath and retport if
// it is changed to changeNotify channel it stops when the ctx.Done() is activated.
// any error during this process is reported to the errCh
func (m *Monitor) watch(ctx context.Context, interval time.Duration, errCh chan<- error, changeNotify chan struct{}) error {
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			f, err := os.Open(m.path)
			if err != nil {
				errCh <- err
			}

			b, err := ioutil.ReadAll(f)
			if err != nil {
				errCh <- err
			}

			newHash := md5.Sum(b)
			if m.first {
				m.lastHash = newHash
				m.first = false
				continue
			}

			for i, b := range m.lastHash {
				if b != newHash[i] {
					changeNotify <- struct{}{}
				}
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
