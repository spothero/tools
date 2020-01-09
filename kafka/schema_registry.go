// Copyright 2020 SpotHero
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kafka

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Shopify/sarama"
	"github.com/linkedin/goavro/v2"
	"github.com/opentracing/opentracing-go"
	"github.com/spf13/pflag"
	shHTTP "github.com/spothero/tools/http"
	"github.com/spothero/tools/log"
	"github.com/spothero/tools/tracing"
)

// SchemaRegistryConfig defines the necessary configuration for interacting with Kafka Schema Registry
type SchemaRegistryConfig struct {
	URL string
}

// RegisterFlags registers schema registry flags with pflags
func (c *SchemaRegistryConfig) RegisterFlags(flags *pflag.FlagSet) {
	flags.StringVar(&c.URL, "kafka-schema-registry-url", "http://localhost:8081", "Kafka schema registry url")
}

// SchemaRegistryClient provides functionality for interacting with Kafka schema registry. This type
// has methods for getting schemas from the registry and decoding sarama ConsumerMessages from Avro into
// Go types. In addition, since the schema registry is immutable, the client contains a cache of schemas
// so that a network request to the regsitry does not have to be made for every Kafka message that needs
// to be decoded.
type SchemaRegistryClient struct {
	SchemaRegistryConfig
	client http.Client
	cache  *sync.Map
}

// NewSchemaRegistryClient creates a schema registry client with the given HTTP metrics bundle.
func (c SchemaRegistryConfig) NewSchemaRegistryClient(httpMetrics shHTTP.Metrics) SchemaRegistryClient {
	retryRoundTripper := shHTTP.RetryRoundTripper{
		RoundTripper: http.DefaultTransport,
		RetriableStatusCodes: map[int]bool{
			http.StatusInternalServerError: true,
			http.StatusBadGateway:          true,
			http.StatusServiceUnavailable:  true,
			http.StatusGatewayTimeout:      true,
		},
		InitialInterval:     10 * time.Millisecond,
		Multiplier:          2,
		MaxInterval:         time.Second,
		RandomizationFactor: 0.5,
		MaxRetries:          5,
	}
	tracingRoundTripper := tracing.RoundTripper{RoundTripper: retryRoundTripper}
	loggingRoundTripper := log.RoundTripper{RoundTripper: tracingRoundTripper}
	metricsRoundTripper := shHTTP.MetricsRoundTripper{RoundTripper: loggingRoundTripper, Metrics: httpMetrics}
	return SchemaRegistryClient{
		SchemaRegistryConfig: c,
		client:               http.Client{Transport: metricsRoundTripper},
		cache:                &sync.Map{},
	}
}

// GetSchema retrieves a textual JSON Avro schema from the Kafka schema registry
func (c SchemaRegistryClient) GetSchema(ctx context.Context, id uint) (string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "get-avro-schema")
	defer span.Finish()
	schema, ok := c.cache.Load(id)
	if ok {
		span.SetTag("outcome", "retrieved_from_cache")
		return schema.(string), nil
	}
	endpoint := fmt.Sprintf("%s/schemas/ids/%d", c.URL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to build schema registry http request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.schemaregistry.v1+json")
	response, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	switch response.StatusCode {
	case http.StatusOK:
		break
	case http.StatusNotFound:
		return "", fmt.Errorf("schema %d not found", id)
	default:
		return "", fmt.Errorf(
			"error while retrieving schema; schema registry returned bad status code %d", response.StatusCode)
	}
	var schemaResponse struct {
		Schema string `json:"schema"`
	}
	if err := json.NewDecoder(response.Body).Decode(&schemaResponse); err != nil {
		return "", err
	}
	c.cache.Store(id, schemaResponse.Schema)
	return schemaResponse.Schema, nil
}

// DecodeKafkaAvroMessage decodes the given Kafka message encoded with Avro into a Go type.
func (c SchemaRegistryClient) DecodeKafkaAvroMessage(ctx context.Context, message *sarama.ConsumerMessage) (interface{}, error) {
	// bytes 1-4 are the schema id (big endian), bytes 5... is the message
	// see: https://docs.confluent.io/current/schema-registry/docs/serializer-formatter.html#wire-format
	if len(message.Value) < 5 {
		return nil, fmt.Errorf("no schema id not found in Kafka message")
	}
	schemaIDBytes := message.Value[1:5]
	messageBytes := message.Value[5:]
	schemaID := binary.BigEndian.Uint32(schemaIDBytes)
	schema, err := c.GetSchema(ctx, uint(schemaID))
	if err != nil {
		return nil, err
	}
	codec, err := goavro.NewCodec(schema)
	if err != nil {
		return nil, err
	}
	decoded, _, err := codec.NativeFromBinary(messageBytes)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}
