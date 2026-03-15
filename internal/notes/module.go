package notes

import "github.com/omakase-dev/go-boilerplate/internal/server"

// Module describes the notes domain.
var Module = server.Module{
	Name:  "notes",
	Path:  "/notes",
	Mount: Mount,
}
