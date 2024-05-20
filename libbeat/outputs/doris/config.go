// Package doris
package doris

import (
	"github.com/elastic/elastic-agent-libs/config"
)

type Config struct {
	FeNodes    []string         `config:"fe_nodes"`
	Username   string           `config:"username"`
	Password   string           `config:"password"`
	Database   string           `config:"database"`
	Table      string           `config:"table"`
	BatchSize  int              `config:"batch_size"`
	MaxRetries int              `config:"max_retries"`
	Config     config.Namespace `config:"config"`
}

var (
	defaultConfig = Config{
		FeNodes:    []string{""},
		Username:   "",
		Password:   "",
		Database:   "",
		Table:      "",
		BatchSize:  1000,
		MaxRetries: 3,
	}
)
