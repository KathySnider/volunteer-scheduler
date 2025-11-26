package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"vol_sched_api/graph"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

func main() {

	// Database connection
	dbURL := os.Getenv("DATABASE_URL")

	if dbURL == "" {
		dbURL = "postgres://postgres:changeme@localhost:5432/volunteer-scheduler?sslmode=disable"
	}

	log.Printf("dbURL: %v", dbURL)

	db, err := sql.Open("postgres", dbURL)
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
