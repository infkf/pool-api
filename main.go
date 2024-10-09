package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DataPoint represents a single record from the pool_usage table
type DataPoint struct {
	ID         int       `json:"id"`
	Timestamp  time.Time `json:"timestamp"`
	Percentage int       `json:"percentage"`
}

// getDatabasePool initializes a connection pool to the PostgreSQL database
func getDatabasePool() (*pgxpool.Pool, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is not set")
	}

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse DATABASE_URL: %v", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %v", err)
	}

	return pool, nil
}

// getDataHandler handles the /data endpoint and returns all data points as JSON
func getDataHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		// Query the database for all data points, ordered by timestamp
		rows, err := pool.Query(context.Background(), "SELECT id, timestamp, percentage FROM pool_usage ORDER BY timestamp")
		if err != nil {
			http.Error(w, "Failed to query the database", http.StatusInternalServerError)
			log.Println("Error querying database:", err)
			return
		}
		defer rows.Close()

		// Collect all data points into a slice
		var dataPoints []DataPoint
		for rows.Next() {
			var dp DataPoint
			err := rows.Scan(&dp.ID, &dp.Timestamp, &dp.Percentage)
			if err != nil {
				http.Error(w, "Failed to scan row", http.StatusInternalServerError)
				log.Println("Error scanning row:", err)
				return
			}
			dataPoints = append(dataPoints, dp)
		}

		// Encode the result as JSON and write to the response
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(dataPoints)
		if err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			log.Println("Error encoding response:", err)
			return
		}
	}
}

func main() {
	// Get a connection pool to the database
	pool, err := getDatabasePool()
	if err != nil {
		log.Fatalf("Error initializing database connection: %v", err)
	}
	defer pool.Close()

	// Set up the HTTP server
	http.HandleFunc("/pool-data", getDataHandler(pool))

	// Start the server
	port := ":8080"
	log.Printf("Starting server on port %s...", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
