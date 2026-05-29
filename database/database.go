package database

import "database/sql"

// Connector opens and returns a live database connection.
// Implementations hold their own configuration (host, port, credentials) so
// the caller only needs to invoke Connect — no parameters required.
// Swap implementations to change the backing database without touching any
// repository or service code.
type Connector interface {
	// Connect opens the connection, configures the pool, and verifies it with a ping.
	Connect() (*sql.DB, error)
}
