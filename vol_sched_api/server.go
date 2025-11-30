package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"

	"vol_sched_api/graph"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

func main() {

	// Database connection

	// Get the postgres password for the database.
	secret, err := os.ReadFile("/run/secrets/secret_db_pw")
	if err != nil {
		log.Fatalf("Unable to read postgres pw: %v", err)

	}
	db_pw := strings.Trim(string(secret), "\n\r")

	// Get the url with a placeholder for the password.
	secret, err = os.ReadFile("/run/secrets/secret_db_url")
	if err != nil {
		log.Fatalf("Unable to read db url: %v", err)
	}
	db_url := strings.Trim(string(secret), "\n\r")

	log.Printf("db_url: %v", db_url)

	// Replace the placeholder with the actual password in the url.
	strings.Replace(db_url, "database_password", db_pw, -1)

	// Connect.
	db, err := sql.Open("postgres", db_url)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	resolver := &graph.Resolver{DB: db}

	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{
		Resolvers: resolver,
	}))

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
	})

	http.Handle("/", c.Handler(playground.Handler("GraphQL playground", "/query")))
	http.Handle("/query", c.Handler(srv))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
