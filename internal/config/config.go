// Package config lê a configuração da aplicação a partir de variáveis de ambiente.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabaseURL string
	HTTPAddr    string
	// CamaraBaseURL permite apontar o ETL para outro host (testes/espelho).
	CamaraBaseURL string
	// CamaraMinInterval é o intervalo mínimo entre requisições à Câmara.
	CamaraMinInterval time.Duration
	// AlertWebhookURL recebe os alertas de falha do ETL (vazio = só log).
	AlertWebhookURL string
	// DailyEvery é o intervalo entre execuções do job diário no agendador.
	DailyEvery time.Duration
}

func FromEnv() (Config, error) {
	c := Config{
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		HTTPAddr:          os.Getenv("HTTP_ADDR"),
		CamaraBaseURL:     os.Getenv("CAMARA_BASE_URL"),
		CamaraMinInterval: 150 * time.Millisecond,
		AlertWebhookURL:   os.Getenv("ALERT_WEBHOOK_URL"),
		DailyEvery:        2 * time.Hour,
	}
	if v := os.Getenv("DAILY_EVERY"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d < time.Minute {
			return Config{}, fmt.Errorf("DAILY_EVERY inválido (use algo como 2h, mínimo 1m): %q", v)
		}
		c.DailyEvery = d
	}
	if v := os.Getenv("CAMARA_MIN_INTERVAL_MS"); v != "" {
		ms, err := strconv.Atoi(v)
		if err != nil || ms < 0 {
			return Config{}, fmt.Errorf("CAMARA_MIN_INTERVAL_MS inválido: %q", v)
		}
		c.CamaraMinInterval = time.Duration(ms) * time.Millisecond
	}
	if c.HTTPAddr == "" {
		c.HTTPAddr = ":8080"
	}
	if c.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL não definido")
	}
	return c, nil
}
