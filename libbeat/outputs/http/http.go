// Package http create by fakzhao
package http

import (
	"errors"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

func init() {
	outputs.RegisterType("http", MakeHTTP)
}

var (
	logger = logp.NewLogger("output.http")
	// ErrNotConnected indicates failure due to client having no valid connection
	ErrNotConnected = errors.New("not connected")
	// ErrJSONEncodeFailed indicates encoding failures
	ErrJSONEncodeFailed = errors.New("json encode failed")
)

func MakeHTTP(
	_ outputs.IndexManager,
	_ beat.Info,
	observer outputs.Observer,
	cfg *conf.C,
) (outputs.Group, error) {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return outputs.Fail(err)
	}
	tlsConfig, err := tlscommon.LoadTLSConfig(config.TLS)
	if err != nil {
		return outputs.Fail(err)
	}
	//hosts, err := outputs.ReadHostList(cfg)
	//if err != nil {
	//	return outputs.Fail(err)
	//}
	proxyURL, err := parseProxyURL(config.ProxyURL)
	if err != nil {
		return outputs.Fail(err)
	}
	if proxyURL != nil {
		logger.Info("Using proxy URL: %s", proxyURL)
	}
	params := config.Params
	if len(params) == 0 {
		params = nil
	}

	logger.Info("Final host URL: " + config.Url)
	var client outputs.NetworkClient
	client, err = NewClient(ClientSettings{
		URL:              config.Url,
		Proxy:            proxyURL,
		TLS:              tlsConfig,
		Username:         config.Username,
		Password:         config.Password,
		Parameters:       params,
		Timeout:          config.Timeout,
		CompressionLevel: config.CompressionLevel,
		Observer:         observer,
		BatchPublish:     config.BatchPublish,
		Headers:          config.Headers,
		ContentType:      config.ContentType,
		Format:           config.Format,
	})

	if err != nil {
		return outputs.Fail(err)
	}
	return outputs.Success(config.Config, config.BatchSize, config.MaxRetries, nil, client)

	//return outputs.Success(config.Config, config.BatchSize, config.MaxRetries, nil, clients)
}
