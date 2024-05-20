// Package doris /
package doris

import (
	"errors"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

func init() {
	outputs.RegisterType("doris", makeDoris)
}

var (
	logger              = logp.NewLogger("output.doris")
	ErrJSONEncodeFailed = errors.New("json encode failed")
)

func makeDoris(
	_ outputs.IndexManager,
	_ beat.Info,
	observer outputs.Observer,
	cfg *conf.C,
) (outputs.Group, error) {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return outputs.Fail(err)
	}
	client, err := NewClient(config)
	if err != nil {
		return outputs.Fail(err)
	}
	return outputs.Success(config.Config, config.BatchSize, config.MaxRetries, nil, client)
}
