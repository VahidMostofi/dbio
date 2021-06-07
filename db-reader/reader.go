package main

import (
	"context"
	"database/sql"
	"log"
	"math/rand"
	"sort"
	"time"
)

type RandomReader struct {
	db     *sql.DB
	rand   *rand.Rand
	events []string
}

// NewRandomReader returns a new RandomReader, seed specifies
// random generator's seed value.
func NewRandomReader(db *sql.DB, seed int64) *RandomReader {
	rr := new(RandomReader)

	rr.db = db
	rr.rand = rand.New(rand.NewSource(seed))

	// list all events
	rr.events = make([]string, 0)
	for event := range RandomGeneratorConstructors {
		rr.events = append(rr.events, event)
	}

	rr.events = sort.StringSlice(rr.events)

	return rr
}

// Read reads from database preiodically.
// It would return if ctx.Done is fed
// Any error occuring during Read operations would be reported to errCh
// Any error reported by the ctx would be returned by the function on exit.
func (rr *RandomReader) Read(ctx context.Context, interval time.Duration, errCh chan<- error) error {
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			event, start, end := rr.generateRandomQueryParameters()
			events, err := event.Retrieve(ctx, start, end, rr.db)
			if err != nil {
				errCh <- err
			}

			log.Printf("retrieved %d records.\n", len(events))
			if len(events) > 0 {
				log.Println(events[0])
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

// generateRandomQueryParameters generates one random event to be used as the
// as the event type that we query. It also returns random start and end for
// executing query.
func (rr *RandomReader) generateRandomQueryParameters() (Event, int64, int64) {

	r := rr.rand.Int()
	// choose one of the events randomly and create an instance
	event := RandomGeneratorConstructors[rr.events[r%len(rr.events)]]()
	start := RandomTimeValue()
	end := RandomTimeValue()

	start, end = Min(start, end), Max(start, end)

	// to cover a larger window, move start to two mintues ago
	start -= 120

	return event, start, end
}

func Min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func Max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
