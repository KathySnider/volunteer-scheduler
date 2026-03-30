package integration

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"volunteer-scheduler/database"
	"volunteer-scheduler/graph/admin"
	adminGen "volunteer-scheduler/graph/admin/generated"
	"volunteer-scheduler/graph/auth"
	authGen "volunteer-scheduler/graph/auth/generated"
	"volunteer-scheduler/graph/volunteer"
	volGen "volunteer-scheduler/graph/volunteer/generated"
	"volunteer-scheduler/middleware"
	"volunteer-scheduler/services"
)

// testServer is the shared httptest.Server used by all tests in this package.
// testDB is the shared database connection for seeding and asserting.
var (
	testServer          *httptest.Server
	testDB              *sql.DB
	testMagicLinkService *services.MagicLinkService
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// -------------------------------------------------------------------------
	// Start a throwaway Postgres container.
	// -------------------------------------------------------------------------
	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:16-alpine"),
		postgres.WithDatabase("volunteer_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		log.Fatalf("Failed to start postgres container: %v", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("Failed to get connection string: %v", err)
	}

	// -------------------------------------------------------------------------
	// Connect and run migrations.
	// Tests run from tests/integration/, so migrations are two levels up.
	// -------------------------------------------------------------------------
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping DB: %v", err)
	}
	testDB = db

	if err := database.RunMigrations(db, "volunteer_test", "file://../../migrations"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// -------------------------------------------------------------------------
	// Set env vars required by services.
	// -------------------------------------------------------------------------
	os.Setenv("APP_URL", "http://localhost:3000")
	os.Setenv("EMAIL_FROM", "test@example.com")
	os.Setenv("SESSION_MAX_AGE", "86400")

	// -------------------------------------------------------------------------
	// Wire up services — same order as main.go.
	// -------------------------------------------------------------------------
	mailer := services.NewTestMailer()
	magicLinkService := services.NewMagicLinkService(db, mailer)
	volunteerService := services.NewVolunteerService(db, mailer)
	shiftService := services.NewShiftService(db, mailer)
	venueService := services.NewVenueService(db)
	feedbackService := services.NewFeedbackService(db, mailer)

	eventService, err := services.NewEventService(db, mailer, shiftService)
	if err != nil {
		log.Fatalf("Failed to create event service: %v", err)
	}

	testMagicLinkService = magicLinkService

	// -------------------------------------------------------------------------
	// Wire up resolvers.
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
		DB:               db,
		EventService:     eventService,
		VolunteerService: volunteerService,
		ShiftService:     shiftService,
		VenueService:     venueService,
		FeedbackService:  feedbackService,
	}

	// -------------------------------------------------------------------------
	// Build the HTTP mux — same routes as main.go, no CORS needed for tests.
	// -------------------------------------------------------------------------
	authSrv := handler.NewDefaultServer(authGen.NewExecutableSchema(authGen.Config{
		Resolvers: authResolver,
	}))
	volunteerSrv := handler.NewDefaultServer(volGen.NewExecutableSchema(volGen.Config{
		Resolvers: volunteerResolver,
	}))
	adminSrv := handler.NewDefaultServer(adminGen.NewExecutableSchema(adminGen.Config{
		Resolvers: adminResolver,
	}))

	mux := http.NewServeMux()
	mux.Handle("/graphql/auth", authSrv)
	mux.Handle("/graphql/volunteer", middleware.RequireAuth(magicLinkService, volunteerSrv))
	mux.Handle("/graphql/admin", middleware.RequireAdmin(magicLinkService, adminSrv))

	testServer = httptest.NewServer(mux)

	// -------------------------------------------------------------------------
	// Run all tests, then clean up.
	// -------------------------------------------------------------------------
	code := m.Run()

	testServer.Close()
	db.Close()
	pgContainer.Terminate(ctx)

	os.Exit(code)
}
