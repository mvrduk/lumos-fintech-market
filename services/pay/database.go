package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitDB() error {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://lumia:lumia_password@localhost:5432/lumia?sslmode=disable"
	}
	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("Error in opening DB: %w", err)
	}

	err = DB.Ping()
	if err != nil {
		return fmt.Errorf("Cannot connect to DB: %w", err)
	}

	log.Println("Payment Service: Connection to DB is Successful")
	return nil
}
