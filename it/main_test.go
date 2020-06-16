package it

import (
	"math/rand"
	"os"
	"testing"
	"time"

	"gitlab.com/gitlab-org/labkit/log"
)

func TestMain(m *testing.M) {
	rand.Seed(time.Now().UnixNano())
	formatter := "text"
	if logFormatter := os.Getenv("TEST_LOG_FORMATTER"); logFormatter != "" {
		formatter = "color"
	}
	// Initialize the global logger
	closer, err := log.Initialize(
		log.WithLogLevel("debug"),
		log.WithOutputName("stderr"),
		log.WithFormatter(formatter),
	)
	if err != nil {
		panic(err)
	}
	code := m.Run()
	closer.Close()
	os.Exit(code)
}
