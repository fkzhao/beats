// Package doris
package doris

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/spf13/cast"
)

type eventRaw map[string]json.RawMessage

type Client struct {
	feNodes      []string
	database     string
	table        string
	observer     outputs.Observer
	encoder      BodyEncoder
	httpClient   *http.Client
	loadBalancer *common.LoadBalancer
}

type Event struct {
	Timestamp string   `json:"timestamp"`
	Time      string   `json:"time"`
	Fields    mapstr.M `json:"-"`
}

func NewClient(cfg Config) (*Client, error) {
	client := &Client{
		feNodes:  cfg.FeNodes,
		database: cfg.Database,
		table:    cfg.Table,
		encoder:  newJSONEncoder(nil),
		httpClient: &http.Client{
			Transport: &http.Transport{
				Proxy: func(req *http.Request) (*url.URL, error) {
					req.SetBasicAuth(cfg.Username, cfg.Password)
					return nil, nil
				},
			},
		},
		loadBalancer: common.NewLoadBalancer(cfg.FeNodes),
	}
	return client, nil
}

func (client *Client) Clone() *Client {
	c, _ := NewClient(
		Config{
			FeNodes: client.feNodes,
			//BatchSize: client.batchSize,
		},
	)
	return c
}

func (client *Client) Connect() error {
	return nil
}

// Close closes a connection.
func (client *Client) Close() error {
	return nil
}

func (client *Client) String() string {
	return "doris output"
}

// Publish sends events to the clients sink.
func (client *Client) Publish(_ context.Context, batch publisher.Batch) error {
	events := batch.Events()
	err := client.publishEvents(events)
	if err == nil {
		batch.ACK()
	} else {
		batch.RetryEvents(events)
	}
	return err
}

func (client *Client) publishEvents(data []publisher.Event) error {
	if len(data) == 0 {
		return nil
	}
	err := client.BatchPublishEvent(data)
	if err != nil {
		return err
	}
	return err
}

func (client *Client) BatchPublishEvent(data []publisher.Event) error {
	var events = make([]eventRaw, len(data))
	for i, event := range data {
		events[i] = makeEvent(&event.Content)
	}
	err := client.submitStreamLoadData(context.Background(), events)
	if err != nil {
		logger.Warn("Fail to insert a single event: %s", err)
		if errors.Is(err, ErrJSONEncodeFailed) {
			return nil
		}
	}
	return err
}

func (client *Client) PublishEvent(data publisher.Event) error {
	event := data
	logger.Debugf("Publish event: %s", event)
	err := client.submitStreamLoadData(context.Background(), makeEvent(&event.Content))
	if err != nil {
		logger.Warn("Fail to insert a single event: %s", err)
	}
	return err
}

func makeEvent(v *beat.Event) map[string]json.RawMessage {
	type event0 Event // prevent recursion
	timeStr := v.Timestamp.UTC().Format("2006-01-02 15:04:05.000")
	e := Event{Timestamp: timeStr, Time: timeStr, Fields: v.Fields}
	b, err := json.Marshal(event0(e))
	if err != nil {
		logger.Warn("Error encoding event to JSON: %v", err)
	}

	var eventMap map[string]json.RawMessage
	err = json.Unmarshal(b, &eventMap)
	if err != nil {
		logger.Warn("Error decoding JSON to map: %v", err)
	}
	// Add the individual fields to the map, flatten "Fields"
	for j, k := range e.Fields {
		b, err = json.Marshal(k)
		if err != nil {
			logger.Warn("Error encoding map to JSON: %v", err)
		}
		eventMap[j] = b
	}
	return eventMap
}

func (client *Client) submitStreamLoadData(ctx context.Context, body interface{}) error {
	if err := client.encoder.Marshal(body); err != nil {
		logger.Warn("Failed to json encode body (%v): %#v", err, body)
		return ErrJSONEncodeFailed
	}
	streamLoadUrl := client.getStreamLoadUrl()
	req, err := http.NewRequestWithContext(ctx, "PUT", streamLoadUrl, client.encoder.Reader())
	if err != nil {
		log.Fatalf("[stream load failed err=%+v]\n", err)
		return err
	}
	req.Header.Add("format", "json")
	req.Header.Add("max_filter_ratio", "1.0")
	req.Header.Add("Label", fmt.Sprintf("olap_fliebeat_stream_load_%v", time.Now().Nanosecond()))
	req.Header.Add("Expect", "100-continue")
	req.Header.Set("strip_outer_array", "true")
	resp, err := client.httpClient.Do(req)

	if err != nil {
		return err
	}

	// Read Response Body
	respBody, _ := io.ReadAll(resp.Body)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("stream load：code=%d, body=%s", resp.StatusCode, string(respBody))
	}

	res := make(map[string]interface{})
	err = json.Unmarshal(respBody, &res)
	if err != nil {
		return err
	}
	// Display Results
	if cast.ToString(res["Status"]) != "Success" {
		return fmt.Errorf("mas=%s, Label = %s", cast.ToString(res["Message"]), cast.ToString(res["Label"]))
	}
	return nil
}

func (client *Client) getStreamLoadUrl() string {
	feNode := client.loadBalancer.GetNextServer().Address
	streamLoadUrl := fmt.Sprintf("http://%s/api/%s/%s/_stream_load", feNode, client.database, client.table)
	return streamLoadUrl
}
