// This is the only file in cmd/app/ that you edit.
// Boilerplate files iterate this module list during startup
// for route mounting, job registration, seed execution, etc.
package main

import (
	"github.com/omaklabs/base/internal/server"
)

// modules lists all domain modules in the app.
// Add a domain: import the package, append its Module here.
// Remove a domain: delete the line and remove the import.
var modules = []server.Module{}
