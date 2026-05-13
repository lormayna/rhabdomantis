package cmd

import (
	"database/sql"
)

type Config struct {
	ShodanAPIKey string
	DBConn       *sql.DB
	Workers      int
}
