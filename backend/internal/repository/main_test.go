package repository

import (
	"os"
	"testing"

	"github.com/oz-fatma/kontrata/backend/internal/mongotest"
)

func TestMain(m *testing.M) {
	os.Exit(mongotest.Run(m))
}
