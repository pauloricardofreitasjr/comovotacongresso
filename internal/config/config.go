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
}

func FromEnv() (Config, error) {
	c := Config{
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		HTTPAddr:          os.Getenv("HTTP_ADDR"),
		CamaraBaseURL:     os.Getenv("CAMARA_BASE_URL"),
		CamaraMinInterval: 150 * time.Millisecond,
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
