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
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

//go:generate go run gen.go

func main() {
	fmt.Println("DB Writer is running...")

	host := readRequireenv("DB_HOST", "")
	user := readRequireenv("DB_USER", "")
	password := readRequireenv("DB_PASSWORD", "")
	dbName := readRequireenv("DB_DATABASE", "")
	port := readRequireenv("DB_PORT", "")

	randomSeed, err := strconv.ParseInt(readRequireenv("RANDOM_SEED", "0"), 10, 64)
	if err != nil {
		log.Fatalf("can't parse RANDOM_SEED: %s", readRequireenv("RANDOM_SEED", "0"))
	}

	writeInterval, err := time.ParseDuration(readRequireenv("WRITE_INTERVAL", "5s"))
	if err != nil {
		log.Fatalf("can't parse WRITE_INTERVAL: %s", readRequireenv("WRITE_INTERVAL", "5s"))
	}

	checkFileInterval, err := time.ParseDuration(readRequireenv("CHECK_INTERVAL", "10s"))
	if err != nil {
		log.Fatalf("can't parse CHECK_INTERVAL: %s", readRequireenv("CHECK_INTERVAL", "10s"))
	}
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC", host, user, password, dbName, port)

	gdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	err = Migrate(gdb)
	if err != nil {
		log.Fatalf("error while migrating database: %s", err.Error())
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// listen for interupt signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	signal.Notify(sigCh, os.Kill)

	ctx, cnF := context.WithCancel(context.Background())

	rw := NewRandomWriter(db, randomSeed)
	errCh := make(chan error)

	go rw.Write(ctx, writeInterval, errCh)

	m := NewMonitor(TypeMappingSource)
	exitToRebornCh := make(chan struct{})
	go m.watch(ctx, checkFileInterval, errCh, exitToRebornCh)

	err = nil
	select {
	case sig := <-sigCh:
		log.Println("Got signal", sig)
		break
	case <-exitToRebornCh:
		log.Println("type map file is changed.")
		cnF()
		db.Close()
		os.Exit(36)
	case err = <-errCh:
		log.Println("error happend")
		break
	}

	cnF()
	if err != nil {
		log.Fatal(err)
	}
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
