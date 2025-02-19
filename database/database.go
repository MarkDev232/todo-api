package database

import (
	"database/sql"
	"fmt"
	"log"
	_ "github.com/lib/pq"
	"sync"
)

var (
	DB     *sql.DB
	once   sync.Once
)

// ConnectDB initializes and returns the database connection, using a singleton pattern.
func ConnectDB() *sql.DB {
	// Ensure that the database connection is initialized only once
	once.Do(func() {
		connStr := "host=localhost port=5432 user=postgres password=root dbname=tododb sslmode=disable"
		var err error
		DB, err = sql.Open("postgres", connStr)
		if err != nil {
			log.Fatalf("Error opening database: %v", err)
		}

		// Ping the database to ensure the connection is valid
		if err = DB.Ping(); err != nil {
			log.Fatalf("Error connecting to the database: %v", err)
		}

		log.Println("Database connection established")
	})

	return DB
}

// CloseDB closes the database connection (should be called when the app exits)
func CloseDB() error {
	if DB != nil {
		return DB.Close()
	}
	return fmt.Errorf("no active database connection")
}

