// Package migrations embeds the SQL files so the migrate command and the
// container image do not depend on the source tree at runtime.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
