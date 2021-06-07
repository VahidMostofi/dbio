package main

import (
	"context"
	"database/sql"
	"math/rand"
	"sort"
	"time"
)

type RandomWriter struct {
	db     *sql.DB
	rand   *rand.Rand
	events []string
}

// NewRandomWriter returns a new RandomWriter, seed specifies
// random generator's seed value.
func NewRandomWriter(db *sql.DB, seed int64) *RandomWriter {
	rw := new(RandomWriter)

	rw.db = db
	rw.rand = rand.New(rand.NewSource(seed))

	// list all events
	rw.events = make([]string, 0)
	for event := range RandomGeneratorConstructors {
		rw.events = append(rw.events, event)
	}

	rw.events = sort.StringSlice(rw.events)

	return rw
}

// Write generates a random event and write it to db.
// It would return if ctx.Done is fed
// Any error occuring during write operations would be reported to errCh
// Any error reported by the ctx would be returned by the function on exit.
func (rw *RandomWriter) Write(ctx context.Context, interval time.Duration, errCh chan<- error) error {
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			event := rw.generateRandomEvent()
			err := event.Store(ctx, rw.db)
			if err != nil {
				errCh <- err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

// generateRandomEvent generates one random event
func (rw *RandomWriter) generateRandomEvent() Event {

	r := rw.rand.Int()
	// choose one of the events randomly and select it's randomInstanceConstructor
	coshenFunction := RandomGeneratorConstructors[rw.events[r%len(rw.events)]]
	event := coshenFunction()
	return event
}
