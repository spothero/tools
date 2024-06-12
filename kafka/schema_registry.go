// Copyright 2023 SpotHero
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
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/linkedin/goavro/v2"
	"github.com/spf13/pflag"
	shHTTP "github.com/spothero/tools/http"
	"github.com/spothero/tools/log"
	"github.com/spothero/tools/tracing"
)

// SchemaRegistryConfig defines the necessary configuration for interacting with Kafka Schema Registry
type SchemaRegistryConfig struct {
	URL string
}

type schemaRequest struct {
	Schema string `json:"schema"`
}

type SchemaResponse struct {
	Subject string `json:"subject"`
	Schema  string `json:"schema"`
	Version int    `json:"version"`
	ID      int    `json:"id"`
}

type errorResponse struct {
	Message   string `json:"message"`
	ErrorCode int    `json:"error_code"`
}

// RegisterFlags registers schema registry flags with pflags
func (c *SchemaRegistryConfig) RegisterFlags(flags *pflag.FlagSet) {
	flags.StringVar(&c.URL, "kafka-schema-registry-url", "http://127.0.0.1:8081", "Kafka schema registry url")
}

// SchemaRegistryProducer defines an interface that contains methods to create schemas and encode kafka messages
// using the uploaded schemas.
type SchemaRegistryProducer interface {
	CreateSchema(ctx context.Context, subject string, schema string, isKey bool) (*SchemaResponse, error)
	EncodeKafkaAvroMessage(ctx context.Context, schemaID uint, message interface{}) ([]byte, error)
}

// SchemaRegistryConsumer defines an interface that contains methods to retrieve schemas and decode kafka messages
// using the retrieved schemas.
type SchemaRegistryConsumer interface {
	GetSchema(ctx context.Context, id uint) (string, error)
	CheckSchema(ctx context.Context, subject string, schema string, isKey bool) (*SchemaResponse, error)
	DecodeKafkaAvroMessage(ctx context.Context, message *sarama.ConsumerMessage) (interface{}, error)
}

// SchemaRegistryClient provides functionality for interacting with Kafka schema registry. This type
// has methods for getting schemas from the registry and decoding sarama ConsumerMessages from Avro into
// Go types. In addition, since the schema registry is immutable, the client contains a cache of schemas
// so that a network request to the registry does not have to be made for every Kafka message that needs
// to be decoded.
type SchemaRegistryClient struct {
	cache  *sync.Map
	client http.Client
	SchemaRegistryConfig
}

// NewSchemaRegistryClient creates a schema registry client with the given HTTP metrics bundle.
func (c SchemaRegistryConfig) NewSchemaRegistryClient(httpMetrics shHTTP.Metrics) *SchemaRegistryClient {
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
	return &SchemaRegistryClient{
		SchemaRegistryConfig: c,
		client:               http.Client{Transport: metricsRoundTripper},
		cache:                &sync.Map{},
	}
}

// Accept header value for the content type expected from the schema registry api
const schemaRegistryAcceptFormat = "application/vnd.schemaregistry.v1+json"

// GetSchema retrieves a textual JSON Avro schema from the Kafka schema registry
func (c *SchemaRegistryClient) GetSchema(ctx context.Context, id uint) (string, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "get-avro-schema")
	defer span.End()

	endpoint := fmt.Sprintf("%s/schemas/ids/%d", c.URL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to build schema registry http request: %w", err)
	}
	req.Header.Set("Accept", schemaRegistryAcceptFormat)
	response, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	switch response.StatusCode {
	case http.StatusOK:
		var decodedResponse SchemaResponse
		if err = json.NewDecoder(response.Body).Decode(&decodedResponse); err != nil {
			return "", err
		}
		return decodedResponse.Schema, nil
	case http.StatusNotFound:
		return "", fmt.Errorf("schema %d not found", id)
	default:
		return "", fmt.Errorf(
			"error while retrieving schema; schema registry returned unhandled status code %d", response.StatusCode)
	}
}

// CheckSchema will check if the schema exists for the given subject
func (c *SchemaRegistryClient) CheckSchema(ctx context.Context, subject string, schema string, isKey bool) (*SchemaResponse, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "check-avro-schema")
	defer span.End()

	concreteSubject := getConcreteSubject(subject, isKey)
	endpoint := fmt.Sprintf("%s/subjects/%s", c.URL, concreteSubject)

	schemaReq := schemaRequest{Schema: schema}
	schemaBytes, err := json.Marshal(schemaReq)
	if err != nil {
		return nil, err
	}
	payload := bytes.NewBuffer(schemaBytes)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to build schema registry http request: %w", err)
	}

	req.Header.Set("Accept", schemaRegistryAcceptFormat)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		var schemaResponse = new(SchemaResponse)
		if err = json.NewDecoder(resp.Body).Decode(schemaResponse); err != nil {
			return nil, err
		}
		return schemaResponse, nil
	case http.StatusNotFound:
		var decodedResponse errorResponse
		if err = json.NewDecoder(resp.Body).Decode(&decodedResponse); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("%s, error code %d", decodedResponse.Message, decodedResponse.ErrorCode)
	default:
		return nil, fmt.Errorf(
			"error while checking schema; schema registry returned unhandled status code %d", resp.StatusCode)
	}
}

// CreateSchema creates a new schema in Schema Registry.
// The Schema Registry compares this against existing known schemas.  If this schema matches an existing schema, a new
// schema will not be created and instead the existing ID will be returned.  This applies even if the schema is assgined
// only to another subject.
func (c *SchemaRegistryClient) CreateSchema(ctx context.Context, subject string, schema string, isKey bool) (*SchemaResponse, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "create-avro-schema")
	defer span.End()

	concreteSubject := getConcreteSubject(subject, isKey)
	schemaReq := schemaRequest{Schema: schema}
	endpoint := fmt.Sprintf("%s/subjects/%s/versions", c.URL, concreteSubject)
	schemaBytes, err := json.Marshal(schemaReq)
	if err != nil {
		return nil, err
	}
	payload := bytes.NewBuffer(schemaBytes)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, payload)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", schemaRegistryAcceptFormat)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		var schemaResponse = new(SchemaResponse)
		if err = json.NewDecoder(resp.Body).Decode(schemaResponse); err != nil {
			return nil, err
		}
		return schemaResponse, nil
	case http.StatusConflict:
		return nil, fmt.Errorf("incompatible schema")
	case http.StatusUnprocessableEntity:
		var decodedResponse errorResponse
		if err = json.NewDecoder(resp.Body).Decode(&decodedResponse); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("%s, error code %d", decodedResponse.Message, decodedResponse.ErrorCode)
	default:
		return nil, fmt.Errorf(
			"error while creating schema; schema registry returned unhandled status code %d", resp.StatusCode)
	}
}

// GetCodec returns an avro codec based on the provided schema id
func (c *SchemaRegistryClient) GetCodec(ctx context.Context, id uint) (*goavro.Codec, error) {
	var codec *goavro.Codec
	codecIface, ok := c.cache.Load(id)

	if ok {
		codec = codecIface.(*goavro.Codec)
	} else {
		schema, err := c.GetSchema(ctx, id)
		if err != nil {
			return nil, err
		}
		codec, err = goavro.NewCodec(schema)
		if err != nil {
			return nil, err
		}
		c.cache.Store(id, codec)
	}

	return codec, nil
}

// DecodeKafkaAvroMessage decodes the given Kafka message encoded with Avro into a Go type.
func (c *SchemaRegistryClient) DecodeKafkaAvroMessage(ctx context.Context, message *sarama.ConsumerMessage) (interface{}, error) {
	// bytes 1-4 are the schema id (big endian), bytes 5... is the message
	// see: https://docs.confluent.io/current/schema-registry/docs/serializer-formatter.html#wire-format
	if len(message.Value) < 5 {
		return nil, fmt.Errorf("no schema id found in Kafka message")
	}

	schemaIDBytes := message.Value[1:5]
	messageBytes := message.Value[5:]
	schemaID := uint(binary.BigEndian.Uint32(schemaIDBytes))

	codec, err := c.GetCodec(ctx, schemaID)
	if err != nil {
		return nil, err
	}

	decoded, _, err := codec.NativeFromBinary(messageBytes)
	if err != nil {
		return nil, err
	}

	return decoded, nil
}

// EncodeKafkaAvroMessage encode the given Kafka message encoded with Avro into a Go type.
func (c *SchemaRegistryClient) EncodeKafkaAvroMessage(ctx context.Context, schemaID uint, message interface{}) ([]byte, error) {
	codec, err := c.GetCodec(ctx, schemaID)
	if err != nil {
		return nil, err
	}

	encoded, err := codec.BinaryFromNative(nil, message)
	if err != nil {
		return nil, err
	}

	return encodeAvro(schemaID, encoded)
}

func getConcreteSubject(subject string, isKey bool) string {
	if isKey {
		subject = fmt.Sprintf("%s-key", subject)
	} else {
		subject = fmt.Sprintf("%s-value", subject)
	}
	return subject
}

// encodeAvro provides the schema registry compliant avro binary
// Notice: the Confluent schema registry has special requirements for the Avro serialization rules,
// not only need to serialize the specific content, but also attach the Schema ID and Magic Byte.
// Ref: https://docs.confluent.io/current/schema-registry/serializer-formatter.html#wire-format
func encodeAvro(schemaID uint, content []byte) ([]byte, error) {
	var binaryMsg []byte
	// Confluent serialization format version number; currently always 0.
	binaryMsg = append(binaryMsg, byte(0))
	binarySchemaID := make([]byte, 4)
	binary.BigEndian.PutUint32(binarySchemaID, uint32(schemaID))
	binaryMsg = append(binaryMsg, binarySchemaID...)
	binaryMsg = append(binaryMsg, content...)
	return binaryMsg, nil
}
