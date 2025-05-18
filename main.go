package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

// Server struct will hold our router (ServeMux) and other server-specific configurations.
type Server struct {
	mux  *http.ServeMux
	addr string
}

type Config struct {
	ListenAddr  string
	PostgresURL string
	Schema      string
}

// NewServer creates and returns a new Server instance.
// It initializes a new ServeMux for routing.
func NewServer(listenAddr string) *Server {
	return &Server{
		mux:  http.NewServeMux(), // Using a new ServeMux allows for more control than DefaultServeMux
		addr: listenAddr,
	}
}

// AddHandlerFunc registers a new handler function for the given pattern.
// This is a convenience method on our Server struct.
func (s *Server) AddHandlerFunc(pattern string, handlerFunc http.HandlerFunc) {
	s.mux.HandleFunc(pattern, handlerFunc)
	log.Printf("Registered handler for pattern: %s", pattern)
}

// AddHandler registers an http.Handler for the given pattern.
// Use this if your handler is a struct implementing http.Handler.
func (s *Server) AddHandler(pattern string, handler http.Handler) {
	s.mux.Handle(pattern, handler)
	log.Printf("Registered handler (http.Handler) for pattern: %s", pattern)
}

// Start begins listening for HTTP requests on the configured address.
func (s *Server) Start() error {
	log.Printf("Server starting on %s...", s.addr)
	// For production, you might want more sophisticated server configuration
	// (e.g., ReadTimeout, WriteTimeout, TLS).
	// http.Server instance allows for this:
	// httpServer := &http.Server{
	//  Addr: s.addr,
	//  Handler: s.mux,
	//  ReadTimeout: 10 * time.Second,
	//  WriteTimeout: 10 * time.Second,
	// }
	// return httpServer.ListenAndServe()

	// For simplicity, we use http.ListenAndServe with our custom mux
	return http.ListenAndServe(s.addr, s.mux)
}

// --- Handler Functions ---

// helloHandler is a simple handler that responds with "Hello, World!".
func helloHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request for %s from %s", r.URL.Path, r.RemoteAddr)
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintln(w, "Hello, World!")
}

func getConfigFromEnv() (Config, error) {
	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":8080"
	}

	postgresURL := os.Getenv("PG_CONN_STRING")
	if postgresURL == "" {
		return Config{}, fmt.Errorf("PG_CONN_STRING environment variable not set")
	}

	schema := os.Getenv("PG_SCHEMA")
	if schema == "" {
		schema = "public"
	}

	return Config{
		ListenAddr:  listenAddr,
		PostgresURL: postgresURL,
		Schema:      schema,
	}, nil
}

func initHTTPServer(cfg Config) *Server {
	// Create a new server instance
	server := NewServer(cfg.ListenAddr)

	// Add the "Hello, World!" handler
	server.AddHandlerFunc("/hello", helloHandler)

	// Add the init handler
	server.AddHandlerFunc("/init", initHandler(cfg))

	// Add the start migration handler
	server.AddHandlerFunc("/start-migration", startMigrationHandler(cfg))

	// Add the complete migration handler
	server.AddHandlerFunc("/complete-migration", completeMigrationHandler(cfg))

	// Add the start and complete migration handler
	server.AddHandlerFunc("/start-and-complete-migration", startAndCompleteMigrationHandler(cfg))

	// Add the rollback handler
	server.AddHandlerFunc("/rollback", rollbackHandler(cfg))

	// Example: Add a handler that responds to the root path "/"
	server.AddHandlerFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// If the path is not exactly "/", it means it wasn't caught by other handlers.
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			log.Printf("404 Not Found for %s", r.URL.Path)
			return
		}
		log.Printf("Received request for %s from %s", r.URL.Path, r.RemoteAddr)
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintln(w, "Welcome to the simple Go HTTP server!")
		fmt.Fprintln(w, "Try /hello or /time")
	})

	return server
}

func main() {
	// Get PostgreSQL connection string from environment variable
	cfg, err := getConfigFromEnv()
	if err != nil {
		log.Fatalf("Failed to get config from env: %v", err)
	}

	// Initialize the server with all routes
	server := initHTTPServer(cfg)

	// Start the server
	// The server will run until an error occurs or the program is terminated.
	err = server.Start()
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
