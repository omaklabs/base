package notes

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/omakase-dev/go-boilerplate/internal/db"
	"github.com/omakase-dev/go-boilerplate/internal/middleware"
	"github.com/omakase-dev/go-boilerplate/internal/server"
	"github.com/omakase-dev/go-boilerplate/internal/testutil"
)

// testUser returns a *db.User suitable for injecting into request context.
func testUser(id int64) *db.User {
	return &db.User{
		ID:        id,
		Email:     "test@example.com",
		Name:      "test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// newAuthenticatedRequest creates a request with the user set in context.
func newAuthenticatedRequest(method, target string, body *strings.Reader, user *db.User) *http.Request {
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, target, body)
	} else {
		r = httptest.NewRequest(method, target, nil)
	}
	return middleware.WithUser(r, user)
}

// setupNotesRouter creates a chi router with note routes wired to the given deps.
// It skips RequireAuth middleware so tests can inject the user directly.
func setupNotesRouter(deps *server.Deps) *chi.Mux {
	r := chi.NewRouter()
	r.Route("/notes", func(r chi.Router) {
		r.Get("/", handleListNotes(deps))
		r.Get("/new", handleNewNote())
		r.Post("/", handleCreateNote(deps))
		r.Get("/{id}", handleShowNote(deps))
		r.Get("/{id}/edit", handleEditNote(deps))
		r.Put("/{id}", handleUpdateNote(deps))
		r.Post("/{id}", handleUpdateNote(deps))
		r.Delete("/{id}", handleDeleteNote(deps))
	})
	return r
}

func TestListNotesEmpty(t *testing.T) {
	_, queries, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	user := testutil.CreateTestUser(t, queries, "alice@example.com")
	deps := &server.Deps{Queries: queries, IsDev: true}
	router := setupNotesRouter(deps)

	req := newAuthenticatedRequest("GET", "/notes", nil, &user)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Notes") {
		t.Error("expected response to contain 'Notes' heading")
	}
}

func TestCreateNote(t *testing.T) {
	_, queries, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	user := testutil.CreateTestUser(t, queries, "alice@example.com")
	deps := &server.Deps{Queries: queries, IsDev: true}
	router := setupNotesRouter(deps)

	form := url.Values{}
	form.Set("title", "My Test Note")
	form.Set("body", "This is the body of the note.")

	req := newAuthenticatedRequest("POST", "/notes", strings.NewReader(form.Encode()), &user)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("expected status 303, got %d", rec.Code)
	}

	loc := rec.Header().Get("Location")
	if !strings.HasPrefix(loc, "/notes/") {
		t.Errorf("expected redirect to /notes/{id}, got %q", loc)
	}

	// Verify the note exists in the database
	count, err := queries.CountNotesByUser(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("CountNotesByUser error: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 note, got %d", count)
	}
}

func TestCreateNoteValidation(t *testing.T) {
	_, queries, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	user := testutil.CreateTestUser(t, queries, "alice@example.com")
	deps := &server.Deps{Queries: queries, IsDev: true}
	router := setupNotesRouter(deps)

	// Submit with empty title
	form := url.Values{}
	form.Set("title", "")
	form.Set("body", "Some body text")

	req := newAuthenticatedRequest("POST", "/notes", strings.NewReader(form.Encode()), &user)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status 422, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "required") {
		t.Error("expected response to contain validation error about 'required'")
	}
}

func TestShowNote(t *testing.T) {
	_, queries, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	user := testutil.CreateTestUser(t, queries, "alice@example.com")

	note, err := queries.CreateNote(context.Background(), db.CreateNoteParams{
		UserID: user.ID,
		Title:  "Test Note",
		Body:   "Body content",
	})
	if err != nil {
		t.Fatalf("CreateNote error: %v", err)
	}

	deps := &server.Deps{Queries: queries, IsDev: true}
	router := setupNotesRouter(deps)

	req := newAuthenticatedRequest("GET", fmt.Sprintf("/notes/%d", note.ID), nil, &user)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Test Note") {
		t.Error("expected response to contain the note title")
	}
}

func TestShowNoteNotFound(t *testing.T) {
	_, queries, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	user := testutil.CreateTestUser(t, queries, "alice@example.com")
	deps := &server.Deps{Queries: queries, IsDev: true}
	router := setupNotesRouter(deps)

	req := newAuthenticatedRequest("GET", "/notes/99999", nil, &user)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func TestDeleteNote(t *testing.T) {
	_, queries, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	user := testutil.CreateTestUser(t, queries, "alice@example.com")

	note, err := queries.CreateNote(context.Background(), db.CreateNoteParams{
		UserID: user.ID,
		Title:  "To Delete",
		Body:   "Will be deleted",
	})
	if err != nil {
		t.Fatalf("CreateNote error: %v", err)
	}

	deps := &server.Deps{Queries: queries, IsDev: true}
	router := setupNotesRouter(deps)

	req := newAuthenticatedRequest("DELETE", fmt.Sprintf("/notes/%d", note.ID), nil, &user)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("expected status 303, got %d", rec.Code)
	}

	// Verify the note is gone
	_, err = queries.GetNote(context.Background(), note.ID)
	if err == nil {
		t.Error("expected error when fetching deleted note, got nil")
	}
}
