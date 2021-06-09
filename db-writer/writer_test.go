package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	_ "github.com/proullon/ramsql/driver"
	"github.com/stretchr/testify/assert"
)

func listEvents(t *testing.T) []string {
	r := make(map[string]interface{})
	b, err := ioutil.ReadFile(TypeMappingSource)
	if err != nil {
		t.Fatalf("cant read file at %s", TypeMappingSource)
	}
	err = json.Unmarshal(b, &r)
	if err != nil {
		t.Fatalf("error while decoding json: %s", err.Error())
	}

	events := make([]string, 0)
	for e := range r {
		events = append(events, e)
	}
	return events
}

func TestShouldMigrateSuccessfully(t *testing.T) {

	db, err := sql.Open("sqlite3", "./test-should-migrate.db")
	if err != nil {
		t.Fatalf("sql.Open : Error : %s\n", err)
	}
	defer db.Close()

	err = Migrate(db, "sqlmock_db_0")
	if err != nil {
		t.Fatalf("error while migrating database: %s", err.Error())
	}

	var count int
	for _, e := range listEvents(t) {
		err = db.QueryRow(fmt.Sprintf("select count(*) from %s;", e)).Scan(&count)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, count, 0)
	}

	err = db.QueryRow("select count(*) from random_name;").Scan(&count)
	assert.NotNil(t, err)
}

func TestShouldWriteSuccessfully(t *testing.T) {

	db, err := sql.Open("sqlite3", "./test.db")
	if err != nil {
		t.Fatalf("sql.Open : Error : %s\n", err)
	}
	defer db.Close()

	err = Migrate(db, "./test.db")
	if err != nil {
		t.Fatalf("error while migrating database: %s", err.Error())
	}

	rw := NewRandomWriter(db, 2)
	errCh := make(chan error)

	writeInterval, _ := time.ParseDuration("200ms")
	waitInterval, _ := time.ParseDuration("3s")
	ticker := time.NewTicker(waitInterval)
	ctx, cnF := context.WithCancel(context.Background())
	go rw.Write(ctx, writeInterval, nil, errCh)
	select {
	case err := <-errCh:
		t.Fatal(err)
	case <-ticker.C:
		cnF()
	}

	var total int
	for _, e := range listEvents(t) {
		var count int
		err = db.QueryRow(fmt.Sprintf("select count(*) from %s;", e)).Scan(&count)
		if err != nil {
			t.Fatal(err)
		}
		total += count
	}
	assert.Greater(t, total, 12)

}

func TestNoDuplicate(t *testing.T) {

	db, err := sql.Open("sqlite3", "./test-no-duplicate.db")
	if err != nil {
		t.Fatalf("sql.Open : Error : %s\n", err)
	}
	defer db.Close()

	err = Migrate(db, "./test-no-duplicate.db")
	if err != nil {
		t.Fatalf("error while migrating database: %s", err.Error())
	}

	eventName := listEvents(t)[0]
	randomConstructor := RandomGeneratorConstructors[eventName]

	// insert a random event twice
	r := rand.New(rand.NewSource(0))
	event := randomConstructor(r)
	err = event.Store(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	err = event.Store(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}

	// count the number of rows in the table
	var count int
	err = db.QueryRow(fmt.Sprintf("select count(*) from %s;", eventName)).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, count)

}
