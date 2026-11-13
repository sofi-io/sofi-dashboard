package migrations

import "embed"

// Files embeds all SQL migration files.
//
//go:embed *.sql
var Files embed.FS
