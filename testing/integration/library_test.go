package integration_test

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/idiomat/dodtnyt/testing/integration"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	tpg "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB
var runIntegrationTests = flag.Bool("integration", false, "run integration tests")

func TestMain(m *testing.M) {
	flag.Parse()

	if *runIntegrationTests {
		ctx := context.Background()
		var err error

		pgUser := "user"
		pgPass := "password"
		pgDB := "test"
		pgc, err := tpg.RunContainer(ctx,
			testcontainers.WithImage("postgres:16-alpine"),
			tpg.WithDatabase(pgDB),
			tpg.WithUsername(pgUser),
			tpg.WithPassword(pgPass),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(5*time.Second)),
		)
		if err != nil {
			slog.Error("failed to start postgres container", "error", err)
			os.Exit(1)
		}

		defer pgc.Terminate(ctx) // nolint:errcheck

		dsn, err := pgc.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			slog.Error("failed to get connection string", "error", err)
			os.Exit(1)
		}

		db, err = gorm.Open(
			postgres.Open(dsn),
			&gorm.Config{},
		)
		if err != nil {
			slog.Error("failed to connect to database", "error", err)
			os.Exit(1)
		}

		if err := db.AutoMigrate(&integration.Author{}, &integration.Book{}); err != nil {
			slog.Error("failed to migrate database", "error", err)
			os.Exit(1)
		}
	}

	// Run the tests
	exitCode := m.Run()

	os.Exit(exitCode)
}

func TestCreateBook_Integration(t *testing.T) {
	if !*runIntegrationTests {
		t.Skip("skipping integration test")
	}

	service := integration.NewLibraryService(db)

	book := &integration.Book{
		Title:   "Meditations",
		Authors: []integration.Author{{Name: "Marcus Aurelius"}},
	}

	err := service.CreateBook(book)

	assert.Nil(t, err)
}

func TestGetBook_Integration(t *testing.T) {
	if !*runIntegrationTests {
		t.Skip("skipping integration test")
	}

	service := integration.NewLibraryService(db)

	book := &integration.Book{
		Title:   "Meditations",
		Authors: []integration.Author{{Name: "Marcus Aurelius"}},
	}

	err := service.CreateBook(book)
	assert.Nil(t, err)

	book, err = service.GetBook(book.ID)
	assert.Nil(t, err)
	assert.Equal(t, "Meditations", book.Title)
	assert.Equal(t, "Marcus Aurelius", book.Authors[0].Name)
}
