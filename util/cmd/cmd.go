// package cmd provides a structured command-style facade to Replicant.
// This package is then used to construct all the various interfaces to Replicant, including
// Programmatic APIs for iOS, Android, C; REST server; CLI, etc.
package cmd

import (
	"github.com/aboodman/replicant/db"
)

type Command interface {
	Run(db *db.DB) error
}
