package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

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
	"github.com/99designs/gqlgen/graphql/handler/transport"
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

// readSecret reads a Docker secret file and trims whitespace.
// Returns an empty string without error if the file does not exist
// (e.g. local development without Docker secrets).
func readSecret(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func main() {
	// -------------------------------------------------------------------------
	// Database connection
	// -------------------------------------------------------------------------

	// Get the database connection URL.
	// In production (Railway, Render, etc.) DATABASE_URL is provided as an env var.
	// Locally with Docker Compose, it is assembled from Docker secret files.
	var db_url string
	if url := os.Getenv("DATABASE_URL"); url != "" {
		db_url = url
	} else {
		secret, err := os.ReadFile("/run/secrets/secret_db_pw")
		if err != nil {
			log.Fatalf("Unable to read postgres pw: %v", err)
		}
		db_pw := strings.TrimSpace(string(secret))

		secret, err = os.ReadFile("/run/secrets/secret_db_url")
		if err != nil {
			log.Fatalf("Unable to read db url: %v", err)
		}
		pattern := strings.TrimSpace(string(secret))
		db_url = strings.Replace(pattern, "database_password", db_pw, -1)
	}

	// Connect.
	db, err := sql.Open("postgres", db_url)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection.
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// -------------------------------------------------------------------------
	// Migrations
	// -------------------------------------------------------------------------

	// In Docker the binary runs from /root/, so migrations are at /app/migrations.
	// Locally, "file://migrations" works if you run from the project root.
	migrationsPath := getEnvWithDefault("MIGRATIONS_PATH", "file:///app/migrations")
	dbName := getEnvWithDefault("DB_NAME", "volunteer-scheduler")
	if err = database.RunMigrations(db, dbName, migrationsPath); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// -------------------------------------------------------------------------
	// Services
	// -------------------------------------------------------------------------

	// Read the Resend API key from an env var (production) or Docker secret file (local).
	// An empty string is fine in local dev — NewMailer uses Mailhog when USE_RESEND is false.
	resendAPIKey := os.Getenv("RESEND_API_KEY")
	if resendAPIKey == "" {
		resendAPIKey = readSecret("/run/secrets/secret_resend_api_key")
	}

	mailer, err := services.NewMailer(resendAPIKey)
	if err != nil {
		log.Fatal("Failed to initialize mailer:", err)
	}

	magicLinkService := services.NewMagicLinkService(db, mailer)
	volunteerService := services.NewVolunteerService(db, mailer)
	shiftService := services.NewShiftService(db, mailer)
	venueService := services.NewVenueService(db)
	feedbackService := services.NewFeedbackService(db, mailer)
	staffService := services.NewStaffService(db)
	fundingEntityService := services.NewFundingEntityService(db)

	eventService, err := services.NewEventService(db, mailer, shiftService)
	if err != nil {
		log.Fatal("Failed to initialize event service:", err)
	}

	// -------------------------------------------------------------------------
	// Background jobs
	// -------------------------------------------------------------------------

	// Run token cleanup once at startup, then every 24 hours.
	go func() {
		if err := magicLinkService.CleanupExpiredTokens(context.Background()); err != nil {
			log.Printf("Initial token cleanup error: %v", err)
		}
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if err := magicLinkService.CleanupExpiredTokens(context.Background()); err != nil {
				log.Printf("Token cleanup error: %v", err)
			}
		}
	}()

	// -------------------------------------------------------------------------
	// Resolvers
	// -------------------------------------------------------------------------

	authResolver := &auth.Resolver{
		MagicLinkService: magicLinkService,
	}

	volunteerResolver := &volunteer.Resolver{
		DB:               db,
		EventService:     eventService,
		VolunteerService: volunteerService,
		ShiftService:     shiftService,
		VenueService:     venueService,
		FeedbackService:  feedbackService,
	}

	adminResolver := &admin.Resolver{
		DB:                   db,
		EventService:         eventService,
		VolunteerService:     volunteerService,
		ShiftService:         shiftService,
		VenueService:         venueService,
		FeedbackService:      feedbackService,
		StaffService:         staffService,
		FundingEntityService: fundingEntityService,
	}

	// -------------------------------------------------------------------------
	// GraphQL servers
	// -------------------------------------------------------------------------

	authSrv := handler.NewDefaultServer(authGen.NewExecutableSchema(authGen.Config{
		Resolvers: authResolver,
	}))

	volunteerSrv := handler.NewDefaultServer(volGen.NewExecutableSchema(volGen.Config{
		Resolvers: volunteerResolver,
	}))
	volunteerSrv.AddTransport(transport.MultipartForm{
		MaxMemory:     32 << 20, // 32 MB RAM buffer per request
		MaxUploadSize: 10 << 20, // 10 MB hard limit (service adds the 5 MB app limit)
	})

	adminSrv := handler.NewDefaultServer(adminGen.NewExecutableSchema(adminGen.Config{
		Resolvers: adminResolver,
	}))
	adminSrv.AddTransport(transport.MultipartForm{
		MaxMemory:     32 << 20, // 32 MB RAM buffer per request
		MaxUploadSize: 10 << 20, // 10 MB hard limit (service adds the 5 MB app limit)
	})

	// -------------------------------------------------------------------------
	// HTTP routing
	// -------------------------------------------------------------------------

	// CORS middleware — allow the frontend origin.
	// FRONTEND_URL can be set explicitly; falls back to APP_URL then localhost.
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = getEnvWithDefault("APP_URL", "http://localhost:3000")
	}
	log.Printf("CORS allowed origin: %s", frontendURL)
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{frontendURL},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	// GraphQL playgrounds (GET, no auth required).
	http.Handle("/auth", playground.Handler("Auth GraphQL", "/graphql/auth"))
	http.Handle("/admin", playground.Handler("Admin GraphQL", "/graphql/admin"))
	http.Handle("/volunteer", playground.Handler("Volunteer GraphQL", "/graphql/volunteer"))

	// GraphQL API endpoints.
	http.Handle("/graphql/auth", c.Handler(authSrv))
	http.Handle("/graphql/volunteer", c.Handler(middleware.RequireAuth(magicLinkService, volunteerSrv)))
	http.Handle("/graphql/admin", c.Handler(middleware.RequireAdmin(magicLinkService, adminSrv)))

	log.Println("Server running on :8080")
	log.Println("Auth endpoint: /graphql/auth")
	log.Println("Volunteer endpoint: /graphql/volunteer")
	log.Println("Admin endpoint: /graphql/admin")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
