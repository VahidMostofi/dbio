package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"time"

	_ "github.com/lib/pq"
)

//go:generate go run gen.go

// Controlling variables related to database connection
var (
	// Details on database connection string
	dbHost     string // maps to DB_HOST
	dbUser     string // maps to DB_USER
	dbPassword string // maps to DB_PASSWORD
	dbName     string // maps to DB_DATABASE
	dbPort     string // amps to DB_PORT

	// number of retires for connecting to db, in case the connection was un-successfull
	maxDBRetryCount int // maps to DB_RETRY_COUNT
)

var (
	// RandomSeed to be used; maps to RANDOM_SEED env
	randomSeed int64

	// how often the reader should read on the database; maps to READ_INTERVAL env
	readInterval time.Duration

	// how often the type mapping file should be monitored for changes; maps to CHECK_INTERVAL env
	checkTypeMappingsInterval time.Duration
)

func main() {
	log.Println("DB reader is starting...")
	initVariables()
	var err error

	// any action that is using ctx as context would be stopped when
	// the cancelFunction is called.
	ctx, cancelFunction := context.WithCancel(context.Background())

	// Create connection to database
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC", dbHost, dbUser, dbPassword, dbName, dbPort)
	db, err := connectToDBWithRetry(ctx, dsn)
	if err != nil {
		log.Fatalf("error while connecting to db: %s", err.Error())
	}
	defer db.Close()

	// Migrate the database to update the schema
	// Migrate is an auto-generated function that updates the schema of database if necessary
	err = Migrate(db, dsn)
	if err != nil {
		log.Fatalf("error while migrating database: %s", err.Error())
	}

	// Instantiate and start the random reader
	rw := NewRandomReader(db, randomSeed)
	errCh := make(chan error)
	go rw.Read(ctx, readInterval, log.Writer(), errCh)

	// Instantiate and start the monitoring code to restart the application if the type mapping
	// is changed.
	// TypeMappingSource is an auto-generated value specifying the path to the type mapping file
	m := NewMonitor(TypeMappingSource)
	exitToRebornCh := make(chan struct{})
	go m.watch(ctx, checkTypeMappingsInterval, errCh, exitToRebornCh)

	// Handle application's lifecyle by listen for interupt signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	signal.Notify(sigCh, os.Kill)

	select {
	case sig := <-sigCh:
		log.Println("Got signal", sig)
		break
	case <-exitToRebornCh:
		log.Println("type map file is changed.")
		cancelFunction()
		db.Close()
		os.Exit(36)
	case err = <-errCh:
		log.Println("error happend during read operations, we stop.")
		break
	}

	// cancel long running tasks
	cancelFunction()

	// if the previous loop stopped due an error, we log it
	if err != nil {
		log.Fatal(err)
	}
}

func initVariables() {
	var err error

	// reading values from environment variables
	dbHost = readRequireenv("DB_HOST", "")
	dbUser = readRequireenv("DB_USER", "")
	dbPassword = readRequireenv("DB_PASSWORD", "")
	dbName = readRequireenv("DB_DATABASE", "")
	dbPort = readRequireenv("DB_PORT", "")

	maxDBRetryCountStr := readRequireenv("DB_RETRY_COUNT", "5")
	maxDBRetryCount, err = strconv.Atoi(maxDBRetryCountStr)
	if err != nil {
		log.Fatalf("can't parse DB_RETRY_COUNT: %s", readRequireenv("DB_RETRY_COUNT", "0"))
	}

	randomSeed, err = strconv.ParseInt(readRequireenv("RANDOM_SEED", "0"), 10, 64)
	if err != nil {
		log.Fatalf("can't parse RANDOM_SEED: %s", readRequireenv("RANDOM_SEED", "0"))
	}

	readInterval, err = time.ParseDuration(readRequireenv("READ_INTERVAL", "5s"))
	if err != nil {
		log.Fatalf("can't parse READ_INTERVAL: %s", readRequireenv("READ_INTERVAL", "5s"))
	}

	checkTypeMappingsInterval, err = time.ParseDuration(readRequireenv("CHECK_INTERVAL", "10s"))
	if err != nil {
		log.Fatalf("can't parse CHECK_INTERVAL: %s", readRequireenv("CHECK_INTERVAL", "10s"))
	}
}

// try connect to db, if failed wait 1 second, next time wait 6 seconds,
// next time wait 11 seconds, ...
// try maxDBRetryCount times
func connectToDBWithRetry(ctx context.Context, dsn string) (*sql.DB, error) {
	var lastSleep = 1
	var err error
	var db *sql.DB
	for i := 0; i < maxDBRetryCount; i++ {

		db, err = sql.Open("postgres", dsn)
		if err != nil {
			d, _ := time.ParseDuration(strconv.Itoa(lastSleep) + "s")
			ticker := time.NewTicker(d)
			log.Println("connecting to database failed, waiting", d, "for reconnecting to database.")
			select {
			case <-ticker.C:
				break
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			d += 5
		}
	}
	if err != nil {
		return nil, err
	}
	return db, nil
}

// readRequireenv read name from os.env, in case none is found:
// if len(defaultValue) > 0, returns defaultValue else panics
func readRequireenv(name string, defaultValue string) string {
	value := os.Getenv(name)
	if len(value) < 1 {
		if len(defaultValue) > 0 {
			return defaultValue
		}
		log.Fatalf("%s needs to be specified in the environment variables.", name)
	}
	return value
}
