package repository

import (
	"os"
	"testing"

	"catgoose/dothog/internal/logger"
)

func TestMain(m *testing.M) {
	logger.Init()
	os.Exit(m.Run())
}
