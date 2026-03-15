# Omakase Go App Rules

## Stack — never deviate
- Go 1.22+
- SQLite via modernc.org/sqlite (no CGO, no Postgres, no MySQL)
- SQLC for database access (no GORM, no raw sql.Query strings in handlers)
- Goose for migrations (plain SQL files only)
- Templ for templates (no html/template, no string concatenation)
- HTMX for server interactivity + Alpine.js for client-side UI (no React, no Vue, no Svelte)
- Tailwind CSS (no other CSS frameworks)
- Chi router (no Gin, no Echo, no Fiber)
- Jobs queue via SQLite jobs table (no Redis, no external queue)

## File conventions
- One handler file per domain: handlers/posts.go, handlers/users.go
- Queries in queries/ as .sql files, never inline in Go
- Templates in templates/ as .templ files, never html/template
- Migrations named: 001_description.sql, 002_description.sql
- Component styling in assets/css/app.css using data-* attribute selectors
- Component variants via data-variant, sizes via data-size, states via data-*
- No global variables except db connection and job queue

## What you must do
- Write SQL in queries/*.sql files, run sqlc generate after
- Write migrations in migrations/*.sql with +goose Up/Down comments
- Run templ generate after adding or changing .templ files
- Add indexes to all foreign key columns
- Use transactions for multi-step writes
- Check errors — never use _ for error returns
- Include CSRF token in all forms (gorilla/csrf provides the template helper)
- Run go test ./... before calling git_commit

## What you must never do
- Use GORM or any ORM
- Write raw SQL strings in handler files
- Use CGO or any library that requires CGO
- Add Node.js, npm, or any JavaScript build tooling
- Use html/template (always use templ)
- Inline styles or non-Tailwind CSS classes
- BEM class variants (use data-* attributes for component variants/sizes/states)
- Use goroutines in handlers (use jobs for async work)
- Skip CSRF tokens on form submissions

## Handler pattern
Every handler: parse input → validate → query db → render template
If HTMX request (isHTMX(r)): render partial only
If full request: render layout + page

## Error handling
User errors (validation): render form with error message, 422
Not found: render 404 page
Server errors: log error, render 500 page, never expose details
Auth required: redirect to /login

## When done
Run: go build ./... (must pass)
Run: go test ./... (must pass)
Then call git_commit with summary
