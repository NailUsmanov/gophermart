package config

import (
	"flag"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Парсим флаги один раз для всех тестов
	flag.Parse()
	os.Exit(m.Run())
}

func TestConfig(t *testing.T) {
	// Сохраняем оригинальные значения флагов
	oldRunAddr := *flagRunAddr
	oldDataBaseURI := *flagDataBaseURI
	oldAccrual := *flagAccural
	defer func() {
		*flagRunAddr = oldRunAddr
		*flagDataBaseURI = oldDataBaseURI
		*flagAccural = oldAccrual
	}()

	t.Run("Default values", func(t *testing.T) {
		os.Clearenv()
		*flagRunAddr = ""
		*flagDataBaseURI = ""
		*flagAccural = ""

		cfg, err := NewConfig()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if cfg.RunAddr != ":8080" {
			t.Errorf("Expected RunAddr :8080, got %s", cfg.RunAddr)
		}
	})

	t.Run("Environment variables", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("RUN_ADDRESS", ":9090")
		defer os.Clearenv()

		*flagRunAddr = ""
		cfg, err := NewConfig()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if cfg.RunAddr != ":9090" {
			t.Errorf("Expected RunAddr :9090, got %s", cfg.RunAddr)
		}
	})

	t.Run("With flags", func(t *testing.T) {
		*flagRunAddr = "9999"
		*flagDataBaseURI = "postgres://test"
		*flagAccural = "http://localhost:9999"

		cfg, err := NewConfig()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if cfg.RunAddr != ":9999" {
			t.Errorf("Expected RunAddr :9999, got %s", cfg.RunAddr)
		}
		if cfg.DataBaseURI != "postgres://test" {
			t.Errorf("Expected DataBaseURI postgres://test, got %s", cfg.DataBaseURI)
		}
		if cfg.Accural != "http://localhost:9999" {
			t.Errorf("Expected Accural http://localhost:9999, got %s", cfg.Accural)
		}
	})
}
