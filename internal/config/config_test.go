package config

import "testing"

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
}
