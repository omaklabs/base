// Package notes is the reference domain implementation.
// Copy this package when creating a new domain, or use
// ./app generate domain <name> to scaffold automatically.
package notes

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/omakase-dev/go-boilerplate/internal/db"
	"github.com/omakase-dev/go-boilerplate/internal/flash"
	"github.com/omakase-dev/go-boilerplate/internal/middleware"
	"github.com/omakase-dev/go-boilerplate/internal/pagination"
	"github.com/omakase-dev/go-boilerplate/internal/server"
	"github.com/omakase-dev/go-boilerplate/internal/validate"
)

// Mount registers all note routes on the given router.
func Mount(r chi.Router, deps *server.Deps) {
	r.Use(middleware.RequireAuth)
	r.Get("/", handleListNotes(deps))
	r.Get("/new", handleNewNote())
	r.Post("/", handleCreateNote(deps))
	r.Get("/{id}", handleShowNote(deps))
	r.Get("/{id}/edit", handleEditNote(deps))
	r.Put("/{id}", handleUpdateNote(deps))
	r.Post("/{id}", handleUpdateNote(deps))
	r.Delete("/{id}", handleDeleteNote(deps))
}

// noteIDFromURL parses the {id} URL parameter as an int64.
func noteIDFromURL(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}

// handleListNotes returns a paginated list of notes for the current user.
func handleListNotes(deps *server.Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := middleware.CurrentUser(r)

		total, err := deps.Queries.CountNotesByUser(r.Context(), user.ID)
		if err != nil {
			server.RenderError(w, r, err, deps.IsDev)
			return
		}

		p := pagination.FromRequest(r, total)

		notes, err := deps.Queries.ListNotes(r.Context(), db.ListNotesParams{
			UserID: user.ID,
			Limit:  int64(p.Limit),
			Offset: int64(p.Offset),
		})
		if err != nil {
			server.RenderError(w, r, err, deps.IsDev)
			return
		}

		if server.IsHTMX(r) {
			NotesListPartial(notes, p, "/notes").Render(r.Context(), w)
			return
		}
		NotesList(notes, p, "/notes").Render(r.Context(), w)
	}
}

// handleShowNote renders a single note by ID.
func handleShowNote(deps *server.Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := noteIDFromURL(r)
		if err != nil {
			server.RenderNotFound(w, r)
			return
		}

		note, err := deps.Queries.GetNote(r.Context(), id)
		if err != nil {
			if err == sql.ErrNoRows {
				server.RenderNotFound(w, r)
				return
			}
			server.RenderError(w, r, err, deps.IsDev)
			return
		}

		NotesShow(note).Render(r.Context(), w)
	}
}

// handleNewNote renders an empty note creation form.
func handleNewNote() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		NotesForm(nil, validate.Errors{}, "", "").Render(r.Context(), w)
	}
}

// handleCreateNote processes the note creation form.
func handleCreateNote(deps *server.Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := middleware.CurrentUser(r)

		if err := r.ParseForm(); err != nil {
			server.RenderError(w, r, err, deps.IsDev)
			return
		}

		title := r.FormValue("title")
		body := r.FormValue("body")

		v := validate.New()
		v.Required("title", title)
		v.MinLength("title", title, 3)

		if v.HasErrors() {
			w.WriteHeader(http.StatusUnprocessableEntity)
			NotesForm(nil, v.Errors(), title, body).Render(r.Context(), w)
			return
		}

		note, err := deps.Queries.CreateNote(r.Context(), db.CreateNoteParams{
			UserID: user.ID,
			Title:  title,
			Body:   body,
		})
		if err != nil {
			server.RenderError(w, r, err, deps.IsDev)
			return
		}

		flash.Set(w, "Note created successfully", "success")
		http.Redirect(w, r, fmt.Sprintf("/notes/%d", note.ID), http.StatusSeeOther)
	}
}

// handleEditNote renders the edit form for an existing note.
func handleEditNote(deps *server.Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := noteIDFromURL(r)
		if err != nil {
			server.RenderNotFound(w, r)
			return
		}

		note, err := deps.Queries.GetNote(r.Context(), id)
		if err != nil {
			if err == sql.ErrNoRows {
				server.RenderNotFound(w, r)
				return
			}
			server.RenderError(w, r, err, deps.IsDev)
			return
		}

		NotesForm(&note, validate.Errors{}, note.Title, note.Body).Render(r.Context(), w)
	}
}

// handleUpdateNote processes the note edit form.
func handleUpdateNote(deps *server.Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := noteIDFromURL(r)
		if err != nil {
			server.RenderNotFound(w, r)
			return
		}

		note, err := deps.Queries.GetNote(r.Context(), id)
		if err != nil {
			if err == sql.ErrNoRows {
				server.RenderNotFound(w, r)
				return
			}
			server.RenderError(w, r, err, deps.IsDev)
			return
		}

		if err := r.ParseForm(); err != nil {
			server.RenderError(w, r, err, deps.IsDev)
			return
		}

		title := r.FormValue("title")
		body := r.FormValue("body")

		v := validate.New()
		v.Required("title", title)
		v.MinLength("title", title, 3)

		if v.HasErrors() {
			w.WriteHeader(http.StatusUnprocessableEntity)
			NotesForm(&note, v.Errors(), title, body).Render(r.Context(), w)
			return
		}

		_, err = deps.Queries.UpdateNote(r.Context(), db.UpdateNoteParams{
			Title: title,
			Body:  body,
			ID:    id,
		})
		if err != nil {
			server.RenderError(w, r, err, deps.IsDev)
			return
		}

		flash.Set(w, "Note updated successfully", "success")
		http.Redirect(w, r, fmt.Sprintf("/notes/%d", id), http.StatusSeeOther)
	}
}

// handleDeleteNote removes a note and redirects to the list.
func handleDeleteNote(deps *server.Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := noteIDFromURL(r)
		if err != nil {
			server.RenderNotFound(w, r)
			return
		}

		if err := deps.Queries.DeleteNote(r.Context(), id); err != nil {
			server.RenderError(w, r, err, deps.IsDev)
			return
		}

		flash.Set(w, "Note deleted successfully", "success")

		// HTMX delete requests: respond with HX-Redirect header
		if server.IsHTMX(r) {
			w.Header().Set("HX-Redirect", "/notes")
			w.WriteHeader(http.StatusOK)
			return
		}

		http.Redirect(w, r, "/notes", http.StatusSeeOther)
	}
}
