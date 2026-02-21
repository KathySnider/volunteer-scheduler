package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"

	"volunteer-scheduler/graph/admin"
	adminGen "volunteer-scheduler/graph/admin/generated"
	"volunteer-scheduler/graph/volunteer"
	volGen "volunteer-scheduler/graph/volunteer/generated"
	"volunteer-scheduler/services"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	_ "github.com/go-sql-driver/mysql"
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
	pattern := strings.Trim(string(secret), "\n\r")

	// Replace the placeholder with the actual password in the url.
	db_url := strings.Replace(pattern, "database_password", db_pw, -1)

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
	defer db.Close()

	// Create services.
	shiftService := services.NewShiftService(db)
	eventService := services.NewEventService(db, shiftService)
	volunteerService := services.NewVolunteerService(db)

	// Create volunteer resolver with services.
	volunteerResolver := &volunteer.Resolver{
		DB:               db,
		EventService:     eventService,
		VolunteerService: volunteerService,
		ShiftService:     shiftService,
	}

	// Create admin resolver with services.
	adminResolver := &admin.Resolver{
		DB:               db,
		EventService:     eventService,
		VolunteerService: volunteerService,
		ShiftService:     shiftService,
	}

	// Set up GraphQL servers with endpoints for user type.
	volunteerSrv := handler.NewDefaultServer(volGen.NewExecutableSchema(volGen.Config{
		Resolvers: volunteerResolver,
	}))

	adminSrv := handler.NewDefaultServer(adminGen.NewExecutableSchema(adminGen.Config{
		Resolvers: adminResolver,
	}))

	// Add CORS middleware
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"}, // Your frontend URL
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	http.Handle("/", playground.Handler("Volunteer GraphQL", "/graphql/volunteer"))
	http.Handle("/admin", playground.Handler("Admin GraphQL", "/graphql/admin"))
	http.Handle("/graphql/volunteer", c.Handler(volunteerSrv))
	http.Handle("/graphql/admin", c.Handler(adminSrv)) // TODO: add AUTH middleware

	log.Println("Server running on :8080")
	log.Println("Volunteer endpoint: /graphql/volunteer")
	log.Println("Admin endpoint: /graphql/admin")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
