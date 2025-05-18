package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/xataio/pgroll/pkg/backfill"
	"github.com/xataio/pgroll/pkg/migrations"
	"github.com/xataio/pgroll/pkg/roll"
	"github.com/xataio/pgroll/pkg/state"
)

func NewRoll(ctx context.Context, postgresURL string, schema string) (*roll.Roll, error) {
	const lockTimeoutMs = 500

	state, err := state.New(ctx, postgresURL, "pgroll")
	if err != nil {
		return nil, err
	}

	roll, err := roll.New(ctx, postgresURL, "public", state, roll.WithLockTimeoutMs(lockTimeoutMs))
	if err != nil {
		return nil, err
	}

	if err := roll.Init(ctx); err != nil {
		return nil, err
	}

	return roll, nil
}

// writeJSONResponse writes a JSON response with the given success status and message
func writeJSONResponse(w http.ResponseWriter, success bool, message string, statusCode int, err error) {
	w.Header().Set("Content-Type", "application/json")
	if statusCode != http.StatusOK {
		w.WriteHeader(statusCode)
	}
	if err != nil {
		fmt.Fprintf(w, `{"success": %t, "message": "%s", "error": "%v"}`, success, message, err)
	} else {
		fmt.Fprintf(w, `{"success": %t, "message": "%s"}`, success, message)
	}
}

// initHandler initializes pgroll with the given configuration
func initHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received request for %s from %s", r.URL.Path, r.RemoteAddr)

		roll, err := NewRoll(context.Background(), cfg.PostgresURL, cfg.Schema)
		if err != nil {
			log.Printf("Failed to initialize pgroll: %v", err)
			writeJSONResponse(w, false, "Failed to initialize pgroll", http.StatusInternalServerError, err)
			return
		}
		defer roll.Close()

		if err := roll.Init(context.Background()); err != nil {
			log.Printf("Failed to initialize pgroll: %v", err)
			writeJSONResponse(w, false, "Failed to initialize pgroll", http.StatusInternalServerError, err)
			return
		}

		writeJSONResponse(w, true, "Successfully initialized pgroll", http.StatusOK, nil)
	}
}

// startMigrationHandler receives a migration JSON and initiates the migration operation
func startMigrationHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received startMigration request for %s from %s", r.URL.Path, r.RemoteAddr)

		if r.Method != http.MethodPost {
			writeJSONResponse(w, false, "Method not allowed", http.StatusMethodNotAllowed, nil)
			return
		}

		// Read the migration JSON from the request body
		defer r.Body.Close()
		var body struct {
			Name       string          `json:"name"`
			Operations json.RawMessage `json:"operations"`
		}
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			log.Printf("Failed to read request body: %v", err)
			writeJSONResponse(w, false, "Failed to read request body", http.StatusBadRequest, err)
			return
		}

		migration, err := migrations.ParseMigration(&migrations.RawMigration{
			Name:       body.Name,
			Operations: body.Operations,
		})
		if err != nil {
			log.Printf("Failed to parse migration: %v", err)
			writeJSONResponse(w, false, "Failed to parse migration", http.StatusBadRequest, err)
			return
		}

		// Initialize roll instance
		roll, err := NewRoll(context.Background(), cfg.PostgresURL, cfg.Schema)
		if err != nil {
			log.Printf("Failed to initialize pgroll: %v", err)
			writeJSONResponse(w, false, "Failed to initialize pgroll", http.StatusInternalServerError, err)
			return
		}
		defer roll.Close()

		// Start the migration
		if err := roll.Start(context.Background(), migration, &backfill.Config{}); err != nil {
			log.Printf("Failed to start migration: %v", err)
			writeJSONResponse(w, false, "Failed to start migration", http.StatusInternalServerError, err)
			return
		}

		writeJSONResponse(w, true, "Migration started successfully", http.StatusOK, nil)
	}
}

// completeMigrationHandler completes a previously started migration
func completeMigrationHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received completeMigration request for %s from %s", r.URL.Path, r.RemoteAddr)

		if r.Method != http.MethodPost {
			writeJSONResponse(w, false, "Method not allowed", http.StatusMethodNotAllowed, nil)
			return
		}

		// Initialize roll instance
		roll, err := NewRoll(context.Background(), cfg.PostgresURL, cfg.Schema)
		if err != nil {
			log.Printf("Failed to initialize pgroll: %v", err)
			writeJSONResponse(w, false, "Failed to initialize pgroll", http.StatusInternalServerError, err)
			return
		}
		defer roll.Close()

		// Complete the migration
		if err := roll.Complete(context.Background()); err != nil {
			log.Printf("Failed to complete migration: %v", err)
			writeJSONResponse(w, false, "Failed to complete migration", http.StatusInternalServerError, err)
			return
		}

		writeJSONResponse(w, true, "Migration completed successfully", http.StatusOK, nil)
	}
}

// startAndCompleteMigrationHandler starts and immediately completes a migration
func startAndCompleteMigrationHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received startAndCompleteMigration request for %s from %s", r.URL.Path, r.RemoteAddr)

		if r.Method != http.MethodPost {
			writeJSONResponse(w, false, "Method not allowed", http.StatusMethodNotAllowed, nil)
			return
		}

		// Read the migration JSON from the request body
		defer r.Body.Close()
		var body struct {
			Name       string          `json:"name"`
			Operations json.RawMessage `json:"operations"`
		}
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			log.Printf("Failed to read request body: %v", err)
			writeJSONResponse(w, false, "Failed to read request body", http.StatusInternalServerError, err)
			return
		}

		migration, err := migrations.ParseMigration(&migrations.RawMigration{
			Name:       body.Name,
			Operations: body.Operations,
		})
		if err != nil {
			log.Printf("Failed to parse migration: %v", err)
			writeJSONResponse(w, false, "Failed to parse migration", http.StatusInternalServerError, err)
			return
		}

		// Initialize roll instance
		roll, err := NewRoll(context.Background(), cfg.PostgresURL, cfg.Schema)
		if err != nil {
			log.Printf("Failed to initialize pgroll: %v", err)
			writeJSONResponse(w, false, "Failed to initialize pgroll", http.StatusInternalServerError, err)
			return
		}
		defer roll.Close()

		// Start the migration
		if err := roll.Start(context.Background(), migration, &backfill.Config{}); err != nil {
			log.Printf("Failed to start migration: %v", err)
			writeJSONResponse(w, false, "Failed to start migration", http.StatusInternalServerError, err)
			return
		}

		// Complete the migration
		if err := roll.Complete(context.Background()); err != nil {
			log.Printf("Failed to complete migration: %v", err)
			writeJSONResponse(w, false, "Failed to complete migration", http.StatusInternalServerError, err)
			return
		}

		writeJSONResponse(w, true, "Migration completed successfully", http.StatusOK, nil)
	}
}

// rollbackHandler rolls back a previously started migration
func rollbackHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received rollback request for %s from %s", r.URL.Path, r.RemoteAddr)

		if r.Method != http.MethodPost {
			writeJSONResponse(w, false, "Method not allowed", http.StatusMethodNotAllowed, nil)
			return
		}

		// Initialize roll instance
		roll, err := NewRoll(context.Background(), cfg.PostgresURL, cfg.Schema)
		if err != nil {
			log.Printf("Failed to initialize pgroll: %v", err)
			writeJSONResponse(w, false, "Failed to initialize pgroll", http.StatusInternalServerError, err)
			return
		}
		defer roll.Close()

		// Rollback the migration
		if err := roll.Rollback(context.Background()); err != nil {
			log.Printf("Failed to rollback migration: %v", err)
			writeJSONResponse(w, false, "Failed to rollback migration", http.StatusInternalServerError, err)
			return
		}

		writeJSONResponse(w, true, "Migration rolled back successfully", http.StatusOK, nil)
	}
}
