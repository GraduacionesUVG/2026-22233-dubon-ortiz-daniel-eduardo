package persistence

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DanielDubon/Gimpy/internal/domain"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const usersTableSchema = `
create table public.users (
  id character varying(255) not null,
  email character varying(255) not null,
  password_hash character varying(255) not null,
  name character varying(255) not null,
  fitness_goals jsonb null default '[]'::jsonb,
  created_at timestamp without time zone null default CURRENT_TIMESTAMP,
  injuries jsonb null default '[]'::jsonb,
  constraint users_pkey primary key (id),
  constraint users_email_key unique (email)
) TABLESPACE pg_default;

create index IF not exists idx_users_email on public.users using btree (email) TABLESPACE pg_default;

create index IF not exists idx_users_created_at on public.users using btree (created_at) TABLESPACE pg_default;
`

// setupTestDB starts an ephemeral PostgreSQL container using testcontainers-go
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("gimpy_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err, "failed to start postgres container")

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "failed to get connection string")

	db, err := sql.Open("pgx", connStr)
	require.NoError(t, err, "failed to open sql connection")

	require.NoError(t, db.PingContext(ctx), "failed to ping database")

	_, err = db.ExecContext(ctx, usersTableSchema)
	require.NoError(t, err, "failed to create users table schema")

	teardown := func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db connection: %v", err)
		}
		if err := pgContainer.Terminate(context.Background()); err != nil {
			t.Logf("failed to terminate postgres container: %v", err)
		}
	}

	return db, teardown
}

func TestPostgresUserRepository_CreateAndFind(t *testing.T) {
	db, teardown := setupTestDB(t)
	defer teardown()

	repo := NewPostgresUserRepository(db)
	ctx := context.Background()

	user := domain.User{
		ID:           "user_test_1",
		Email:        "create-and-find@example.com",
		PasswordHash: "hashed-password",
		Name:         "Integration Test User",
		FitnessGoals: []string{"lose weight", "build muscle"},
		Injuries: []domain.Injury{
			{Description: "knee pain", ReportedAt: time.Now().Truncate(time.Second)},
		},
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err, "Create should not return an error")

	found, err := repo.FindByEmail(ctx, user.Email)
	require.NoError(t, err, "FindByEmail should not return an error")

	assert.Equal(t, user.ID, found.ID)
	assert.Equal(t, user.Name, found.Name)
	assert.Equal(t, user.Email, found.Email)
	assert.Equal(t, user.PasswordHash, found.PasswordHash)
	assert.ElementsMatch(t, user.FitnessGoals, found.FitnessGoals)
	require.Len(t, found.Injuries, 1)
	assert.Equal(t, user.Injuries[0].Description, found.Injuries[0].Description)
}

func TestPostgresUserRepository_DuplicateEmail(t *testing.T) {
	db, teardown := setupTestDB(t)
	defer teardown()

	repo := NewPostgresUserRepository(db)
	ctx := context.Background()

	firstUser := domain.User{
		ID:           "user_test_2",
		Email:        "duplicate@example.com",
		PasswordHash: "hashed-password",
		Name:         "First User",
		FitnessGoals: []string{"lose weight"},
	}

	err := repo.Create(ctx, firstUser)
	require.NoError(t, err, "first Create should succeed")

	secondUser := domain.User{
		ID:           "user_test_3",
		Email:        "duplicate@example.com", // same email
		PasswordHash: "another-hashed-password",
		Name:         "Second User",
		FitnessGoals: []string{"build muscle"},
	}

	err = repo.Create(ctx, secondUser)
	require.Error(t, err, "second Create with duplicate email should return an error")
	assert.ErrorIs(t, err, domain.ErrEmailAlreadyExists)
}
