package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
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

	// list all events using RandomGeneratorConstructors that is an auto-generated map
	// from event name to a function that generates random instance of that event
	rw.events = make([]string, 0)
	for event := range RandomGeneratorConstructors {
		rw.events = append(rw.events, event)
	}

	// because the events are created based on a map
	// we need to ensure they have the same order if
	// it is computed multiple times over different
	// instances
	rw.events = sort.StringSlice(rw.events)

	return rw
}

// Write generates a random event and write it to db.
// It would return if ctx.Done is fed
// Any error occuring during write operations would be reported to errCh
// Any error reported by the ctx would be returned by the function on exit.
func (rw *RandomWriter) Write(ctx context.Context, interval time.Duration, statsWriter io.Writer, errCh chan<- error) error {
	ticker := time.NewTicker(interval)
	logTicker := time.NewTicker(time.Duration(10) * time.Second)
	counter := 0
	for {
		select {
		case <-ticker.C:
			event := rw.generateRandomEvent()
			err := event.Store(ctx, rw.db)
			if err != nil {
				errCh <- err
			}
			counter++
		case <-logTicker.C:
			if statsWriter != nil {
				fmt.Fprintf(statsWriter, "wrote %d events in the past 10 seconds.\n", counter)
			}
			counter = 0
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
	event := coshenFunction(rw.rand)
	return event
}
