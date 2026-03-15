package notes

import "github.com/omaklabs/base/internal/server"

// Module describes the notes domain.
var Module = server.Module{
	Name:  "notes",
	Path:  "/notes",
	Mount: Mount,
}
