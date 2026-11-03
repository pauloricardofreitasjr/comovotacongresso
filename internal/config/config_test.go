package config

import (
	"testing"
	"time"
)

func TestFromEnv(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://x")
		t.Setenv("HTTP_ADDR", "")
		c, err := FromEnv()
		if err != nil || c.DatabaseURL != "postgres://x" || c.HTTPAddr != ":8080" {
			t.Fatalf("c=%+v err=%v", c, err)
		}
	})
	t.Run("addr customizado", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://x")
		t.Setenv("HTTP_ADDR", ":9000")
		if c, _ := FromEnv(); c.HTTPAddr != ":9000" {
			t.Fatalf("addr=%q", c.HTTPAddr)
		}
	})
	t.Run("sem DATABASE_URL", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "")
		if _, err := FromEnv(); err == nil {
			t.Fatal("esperava erro")
		}
	})
	t.Run("intervalo da Câmara", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://x")
		t.Setenv("CAMARA_MIN_INTERVAL_MS", "")
		if c, _ := FromEnv(); c.CamaraMinInterval != 150*time.Millisecond {
			t.Fatalf("default=%v", c.CamaraMinInterval)
		}
		t.Setenv("CAMARA_MIN_INTERVAL_MS", "400")
		if c, _ := FromEnv(); c.CamaraMinInterval != 400*time.Millisecond {
			t.Fatalf("custom=%v", c.CamaraMinInterval)
		}
		t.Setenv("CAMARA_MIN_INTERVAL_MS", "abc")
		if _, err := FromEnv(); err == nil {
			t.Fatal("esperava erro")
		}
	})
}

func TestFromEnv_DailyEvery(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	c, err := FromEnv()
	if err != nil || c.DailyEvery != 2*time.Hour {
		t.Fatalf("padrão: %v %v", c.DailyEvery, err)
	}
	t.Setenv("DAILY_EVERY", "30m")
	if c, err = FromEnv(); err != nil || c.DailyEvery != 30*time.Minute {
		t.Fatalf("30m: %v %v", c.DailyEvery, err)
	}
	for _, ruim := range []string{"2", "abc", "10s"} {
		t.Setenv("DAILY_EVERY", ruim)
		if _, err := FromEnv(); err == nil {
			t.Errorf("DAILY_EVERY=%q deveria ser inválido", ruim)
		}
	}
}
