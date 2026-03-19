package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"

	"volunteer-scheduler/database"
	"volunteer-scheduler/graph/admin"
	adminGen "volunteer-scheduler/graph/admin/generated"
	"volunteer-scheduler/graph/auth"
	authGen "volunteer-scheduler/graph/auth/generated"
	"volunteer-scheduler/graph/volunteer"
	volGen "volunteer-scheduler/graph/volunteer/generated"
	"volunteer-scheduler/middleware"
	"volunteer-scheduler/services"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

func getEnvWithDefault(key, fallback string) string {
	val, ok := os.LookupEnv(key)
	if ok {
		return val
	} else {
		return fallback
	}
}

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

	// Run migrations.
	// In Docker the binary runs from /root/, so migrations are at /app/migrations.
	// Locally, "file://migrations" works if you run from the project root.
	//migrationsPath := getEnvWithDefault("MIGRATIONS_PATH", "file://migrations")
	migrationsPath := getEnvWithDefault("MIGRATIONS_PATH", "file:///app/migrations")
	dbName := getEnvWithDefault("DB_NAME", "volunteer-scheduler")
	err = database.RunMigrations(db, dbName, migrationsPath)
	if err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Create services.
	mailer, err := services.NewMailer()
	if err != nil {
		log.Fatal("Failed to initialize mailer:", err)
	}
	magicLinkService := services.NewMagicLinkService(db, mailer)
	volunteerService := services.NewVolunteerService(db, mailer)
	shiftService := services.NewShiftService(db, mailer)
	venueService := services.NewVenueService(db)

	eventService, err := services.NewEventService(db, shiftService)
	if err != nil {
		log.Fatal("Failed to initialize event service:", err)
	}

	// Create the resolvers with services.
	authResolver := &auth.Resolver{
		MagicLinkService: magicLinkService,
	}

	volunteerResolver := &volunteer.Resolver{
		DB:               db,
		EventService:     eventService,
		VolunteerService: volunteerService,
		ShiftService:     shiftService,
		VenueService:     venueService,
	}

	adminResolver := &admin.Resolver{
		DB:               db,
		EventService:     eventService,
		VolunteerService: volunteerService,
		ShiftService:     shiftService,
		VenueService:     venueService,
	}

	// Set up GraphQL servers with endpoints for user type.
	authSrv := handler.NewDefaultServer(authGen.NewExecutableSchema(authGen.Config{
		Resolvers: authResolver,
	}))

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

	http.Handle("/auth", playground.Handler("Auth GraphQL", "/graphql/auth"))
	http.Handle("/admin", playground.Handler("Admin GraphQL", "/graphql/admin"))
	http.Handle("/volunteer", playground.Handler("Volunteer GraphQL", "/graphql/volunteer"))
	http.Handle("/graphql/auth", c.Handler(authSrv))
	http.Handle("/graphql/volunteer", c.Handler(middleware.RequireAuth(magicLinkService, volunteerSrv)))
	http.Handle("/graphql/admin", c.Handler(middleware.RequireAdmin(magicLinkService, adminSrv)))

	log.Println("Server running on :8080")
	log.Println("Auth endpoint: /graphql/auth")
	log.Println("Volunteer endpoint: /graphql/volunteer")
	log.Println("Admin endpoint: /graphql/admin")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
