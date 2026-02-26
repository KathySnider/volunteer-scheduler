package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"

	"volunteer-scheduler/graph/volunteer"
	"volunteer-scheduler/graph/volunteer/generated"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

func main() {

	// Database connection
	var dbURL string

	// First, try DATABASE_URL env var (for local dev without Docker secrets)
	if envURL := os.Getenv("DATABASE_URL"); envURL != "" {
		dbURL = envURL
		log.Println("Using DATABASE_URL from environment")
	} else {
		// Fall back to Docker secrets (for container deployment)
		secret, err := os.ReadFile("/run/secrets/secret_db_pw")
		if err != nil {
			log.Fatalf("Unable to read postgres pw (set DATABASE_URL for local dev): %v", err)
		}
		dbPw := strings.Trim(string(secret), "\n\r")

		secret, err = os.ReadFile("/run/secrets/secret_db_url")
		if err != nil {
			log.Fatalf("Unable to read db url (set DATABASE_URL for local dev): %v", err)
		}
		pattern := strings.Trim(string(secret), "\n\r")

		dbURL = strings.Replace(pattern, "database_password", dbPw, -1)
	}

	// Connect.
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	resolver := &volunteer.Resolver{DB: db}

	volunteerHandler := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{
		Resolvers: resolver,
	}))

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
	})

	http.Handle("/", c.Handler(playground.Handler("GraphQL playground", "/query")))
	http.Handle("/query", c.Handler(volunteerHandler))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
