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
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/linkedin/goavro/v2"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSchemaRegistryConfig_RegisterFlags(t *testing.T) {
	flags := pflag.NewFlagSet("pflags", pflag.PanicOnError)
	c := SchemaRegistryConfig{}
	c.RegisterFlags(flags)
	err := flags.Parse([]string{"--kafka-schema-registry-url", "http://schema.registry"})
	require.NoError(t, err)
	assert.Equal(t, "http://schema.registry", c.URL)
}

type mockTransport struct {
	mock.Mock
	t *testing.T
}

const schemaID = uint(77)

func (m *mockTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	call := m.Called(request)
	m.t.Helper()
	assert.Equal(m.t, "application/vnd.schemaregistry.v1+json", request.Header.Get("Accept"))
	assert.Equal(m.t, "schema.registry/schemas/ids/77", request.URL.String())
	return call.Get(0).(*http.Response), call.Error(1)
}

func TestSchemaRegistryClient_GetSchema(t *testing.T) {
	tests := []struct {
		name              string
		url               string
		registryResponse  *http.Response
		httpErr           bool
		expectHTTPRequest bool
		expectedSchema    string
		expectErr         bool
	}{
		{
			"schema is retrieved from schema registry",
			"schema.registry",
			&http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader("{\"schema\": \"it's a schema\"}")),
			},
			false,
			true,
			"it's a schema",
			false,
		}, {
			"http error returns error",
			"schema.registry",
			&http.Response{},
			true,
			true,
			"",
			true,
		}, {
			"404 returns error",
			"schema.registry",
			&http.Response{StatusCode: http.StatusNotFound},
			false,
			true,
			"",
			true,
		}, {
			"non-200 returns error",
			"schema.registry",
			&http.Response{StatusCode: http.StatusTeapot},
			false,
			true,
			"",
			true,
		}, {
			"bad json returns error",
			"schema.registry",
			&http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader("not json")),
			},
			false,
			true,
			"",
			true,
		}, {
			"error building request returns error",
			"💀://",
			nil,
			false,
			false,
			"",
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := SchemaRegistryClient{
				SchemaRegistryConfig: SchemaRegistryConfig{URL: test.url},
				client:               http.Client{Transport: &mockTransport{t: t}},
				cache:                &sync.Map{},
			}
			if test.expectHTTPRequest {
				mockCall := client.client.Transport.(*mockTransport).On("RoundTrip", mock.Anything)
				if test.httpErr {
					mockCall.Return(test.registryResponse, fmt.Errorf("http error"))
				} else {
					mockCall.Return(test.registryResponse, nil)
				}
				defer client.client.Transport.(*mockTransport).AssertExpectations(t)
			}
			schema, err := client.GetSchema(context.Background(), schemaID)
			if test.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, test.expectedSchema, schema)
		})
	}
}

const avroSchema = `{"type": "record", "name": "test", "fields": [{"name": "name", "type": "string"}]}`

func newAvroMessage(t *testing.T) *sarama.ConsumerMessage {
	t.Helper()
	codec, err := goavro.NewCodec(avroSchema)
	require.Nil(t, err)
	avroMessage := map[string]interface{}{"name": "Guy Fieri"}
	avroBytes, err := codec.BinaryFromNative(nil, avroMessage)
	require.Nil(t, err)

	// Construct a fake Kafka message using the Kafka schema wire format
	fakeKafkaMessage := []byte{0, 0, 0, 0, 77}
	fakeKafkaMessage = append(fakeKafkaMessage, avroBytes...)
	return &sarama.ConsumerMessage{Value: fakeKafkaMessage}
}

func TestSchemaRegistryClient_DecodeKafkaAvroMessage(t *testing.T) {
	tests := []struct {
		name             string
		msg              *sarama.ConsumerMessage
		prePopulateCache bool
		schema           string
		expected         interface{}
		expectErr        bool
	}{
		{
			"avro message is decoded",
			newAvroMessage(t),
			false,
			avroSchema,
			map[string]interface{}{"name": "Guy Fieri"},
			false,
		}, {
			"too short of a message returns error",
			&sarama.ConsumerMessage{Value: []byte{1, 2}},
			false,
			"",
			nil,
			true,
		}, {
			"error getting schema returns error",
			newAvroMessage(t),
			false,
			"",
			nil,
			true,
		}, {
			"error creating avro codec returns error",
			newAvroMessage(t),
			false,
			"bad schema",
			nil,
			true,
		}, {
			"error decoding message returns error",
			&sarama.ConsumerMessage{Value: []byte{0, 0, 0, 0, 77, 78, 79}}, // junk message data with correct schema id
			false,
			avroSchema,
			nil,
			true,
		}, {
			"codec already in cache uses it",
			newAvroMessage(t),
			true,
			avroSchema,
			map[string]interface{}{"name": "Guy Fieri"},
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := SchemaRegistryClient{
				SchemaRegistryConfig: SchemaRegistryConfig{URL: "schema.registry"},
				cache:                &sync.Map{},
				client:               http.Client{Transport: &mockTransport{t: t}},
			}
			if test.prePopulateCache {
				codec, err := goavro.NewCodec(test.schema)
				require.NoError(t, err)
				client.cache.Store(schemaID, codec)
			} else {
				getSchema := client.client.Transport.(*mockTransport).On("RoundTrip", mock.Anything)
				if test.schema != "" {
					getSchema.Return(&http.Response{
						StatusCode: http.StatusOK,
						Body: ioutil.NopCloser(
							strings.NewReader(
								fmt.Sprintf("{\"schema\": \"%s\"}", strings.Replace(test.schema, "\"", "\\\"", -1)))),
					}, nil)
				} else {
					getSchema.Return(&http.Response{StatusCode: http.StatusInternalServerError}, nil)
				}
			}
			outcome, err := client.DecodeKafkaAvroMessage(context.Background(), test.msg)
			if test.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, test.expected, outcome)
		})
	}
}
