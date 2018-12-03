// Copyright 2018 SpotHero
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

package tools

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"sync"

	"github.com/Shopify/sarama"
	"github.com/linkedin/goavro"
	"github.com/opentracing/opentracing-go"
	"go.uber.org/zap"
)

type kafkaSchemaRegistryClient interface {
	getSchema(ctx context.Context, schemaID int, schemaRegistryURL string) (string, error)
}

type schemaRegistryClient struct{}

// SchemaRegistryConfig defines the necessary configuration for interacting with Schema Registry
type SchemaRegistryConfig struct {
	SchemaRegistryURL  string
	schemas            sync.Map
	client             kafkaSchemaRegistryClient
	messageUnmarshaler kafkaMessageUnmarshaler
}

func (src *schemaRegistryClient) getSchema(ctx context.Context, schemaID int, schemaRegistryURL string) (string, error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "get-avro-schema")
	defer span.Finish()
	endpoint := fmt.Sprintf("%s/schemas/ids/%d", schemaRegistryURL, schemaID)
	response, err := http.Get(endpoint)
	if err != nil {
		Logger.Error(
			"Error getting schema from schema registry",
			zap.Int("schema_id", schemaID), zap.Error(err))
		return "", err
	}
	var schemaResponse struct {
		Schema *string `json:"schema"`
	}
	if err := json.NewDecoder(response.Body).Decode(&schemaResponse); err != nil {
		return "", err
	}
	return *schemaResponse.Schema, nil
}

func (src *SchemaRegistryConfig) unmarshalMessage(ctx context.Context, message []byte, target interface{}) []error {
	// bytes 1-4 are the schema id (big endian), bytes 5... is the message
	// see: https://docs.confluent.io/current/schema-registry/docs/serializer-formatter.html#wire-format
	span, ctx := opentracing.StartSpanFromContext(ctx, "get-avro-schema")
	defer span.Finish()
	schemaIDBytes := message[1:5]
	messageBytes := message[5:]
	schemaID := binary.BigEndian.Uint32(schemaIDBytes)

	// Check if the schema is in cache, if not, get it from schema registry and
	// cache it
	schemaInterface, schemaInCache := src.schemas.Load(schemaID)
	var schema string
	if !schemaInCache {
		Logger.Info(
			"Schema not in cache, requesting from schema schema registry and caching",
			zap.Uint32("schema_id", schemaID))
		var getSchemaErr error
		schema, getSchemaErr = src.client.getSchema(ctx, int(schemaID), src.SchemaRegistryURL)
		if getSchemaErr != nil {
			Logger.Error(
				"Error getting schema from schema schemaRegistry",
				zap.Error(getSchemaErr), zap.Uint32("schema_id", schemaID))
			return []error{getSchemaErr}
		}
		src.schemas.Store(schemaID, schema)
	} else {
		schema = schemaInterface.(string)
	}

	// Decode Avro from binary
	codec, codecErr := goavro.NewCodec(schema)
	if codecErr != nil {
		return []error{codecErr}
	}
	decoded, _, decodeErr := codec.NativeFromBinary(messageBytes)
	if decodeErr != nil {
		return []error{decodeErr}
	}

	// Unmarshal avro to Go type
	return src.messageUnmarshaler.unmarshalKafkaMessageMap(decoded.(map[string]interface{}), target)
}

// UnmarshalMessage Implements the KafkaMessageUnmarshaler interface.
// Decodes an Avro message into a Go struct type, specifically an Avro message
// from Kafka. Avro schemas are fetched from Kafka schema registry.
// To use this function, tag each field of the target struct with a `kafka`
// tag whose value indicates which key on the Avro message to set as the
// value.
func (src *SchemaRegistryConfig) UnmarshalMessage(
	ctx context.Context,
	msg *sarama.ConsumerMessage,
	target interface{},
) error {
	unmarshalErrs := src.unmarshalMessage(ctx, msg.Value, target)
	if len(unmarshalErrs) > 0 {
		Logger.Error(
			"Unable to unmarshal from Avro", zap.Errors("errors", unmarshalErrs),
			zap.String("type", reflect.TypeOf(target).String()))
		return fmt.Errorf("unable to unmarshal from Avro")
	}
	return nil
}
