package testutil

import (
	"os"
)

var testDatabase = os.Getenv("VISOR_TEST_DB")

// DatabaseAvailable reports whether a database is available for testing
func DatabaseAvailable() bool {
	return testDatabase != ""
}

// Database returns the connection string for connecting to the test database
func Database() string {
	return testDatabase
}
