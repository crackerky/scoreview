package store_test

import (
	"os"
	"testing"

	"scoreview/api/internal/testdb"
)

func TestMain(m *testing.M) { os.Exit(testdb.Run(m)) }
