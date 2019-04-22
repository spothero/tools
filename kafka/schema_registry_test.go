// Copyright 2019 SpotHero
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
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"testing"

	"github.com/linkedin/goavro"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock out http transport
type mockTransport struct {
	mock.Mock
	Response *http.Response
	Err      error
}

func (m *mockTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	m.Called(request)
	return m.Response, m.Err
}

func TestGetSchema(t *testing.T) {
	ctx := context.Background()
	var httpDefaultClient *http.Client
	// need to mutex the http client because this is a global
	httpClientMutex := &sync.Mutex{}

	setUp := func(transport *mockTransport) {
		// Save the http default client to restore state after tests complete
		httpDefaultClient = http.DefaultClient
		// Set the http transport to our mocked transport
		transport.On("RoundTrip", mock.Anything)
		httpClientMutex.Lock()
		http.DefaultClient = &http.Client{Transport: transport}
	}

	tearDown := func() {
		// Restore the http default client
		http.DefaultClient = httpDefaultClient
		httpClientMutex.Unlock()
	}

	// Test that schemas are retrieved from schema registry
	t.Run("retrieve schemas from Kafka schema registry", func(t *testing.T) {
		defer tearDown()
		transport := &mockTransport{
			Response: &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(bytes.NewBufferString("{\"schema\": \"test\"}")),
			},
			Err: nil,
		}
		setUp(transport)
		client := &schemaRegistryClient{}
		schema, err := client.getSchema(ctx, 2, "http://schema.registry")
		assert.Nil(t, err)
		assert.Equal(t, "test", schema)
		require.Len(t, transport.Calls, 1)
		require.Len(t, transport.Calls[0].Arguments, 1)
		request := transport.Calls[0].Arguments[0].(*http.Request)
		assert.Equal(t, request.URL.Host, "schema.registry")
		assert.Equal(t, request.URL.Path, "/schemas/ids/2")
	})

	// Test handling errors from schema registry
	t.Run("handle schema registry errors", func(t *testing.T) {
		defer tearDown()
		transport := &mockTransport{
			Response: nil,
			Err:      fmt.Errorf("something happened"),
		}
		setUp(transport)
		client := &schemaRegistryClient{}
		schema, err := client.getSchema(ctx, 2, "http://schema.registry")
		assert.Empty(t, schema)
		assert.Contains(t, err.Error(), "something happened")
	})

	// Test handling invalid JSON returned from schema registry
	t.Run("handle json decode errors", func(t *testing.T) {
		defer tearDown()
		transport := &mockTransport{
			Response: &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(bytes.NewBufferString("{\"schema\": \"invalid json}")),
			},
			Err: nil,
		}
		setUp(transport)
		client := &schemaRegistryClient{}
		schema, err := client.getSchema(ctx, 2, "http://schema.registry")
		assert.Empty(t, schema)
		assert.Equal(t, "unexpected EOF", err.Error())
	})
}

// Implement a mocked out schema registry client
type SchemaRegistryClientMock struct {
	mock.Mock
}

func (sm *SchemaRegistryClientMock) getSchema(ctx context.Context, schemaID int, _ string) (string, error) {
	args := sm.Called(schemaID, ctx)
	return args.String(0), args.Error(1)
}

func (sm *SchemaRegistryClientMock) unmarshalKafkaMessageMap(kafkaMessageMap map[string]interface{}, target interface{}) []error {
	args := sm.Called(kafkaMessageMap)
	return args.Get(0).([]error)
}

func setupMockSchemaRegistry(t *testing.T) (*SchemaRegistryConfig, *SchemaRegistryClientMock, string, []byte, map[string]interface{}) {
	mockClient := &SchemaRegistryClientMock{}
	config := &SchemaRegistryConfig{client: mockClient, messageUnmarshaler: mockClient}

	// Construct an Avro message to use for testing
	avroSchema := `
			{
				"type": "record",
				"name": "test",
				"fields": [{"name": "name", "type": "string"}]
			}`
	codec, err := goavro.NewCodec(avroSchema)
	require.Nil(t, err)
	avroMessage := map[string]interface{}{"name": "Guy Fieri"}
	avroBytes, err := codec.BinaryFromNative(nil, avroMessage)
	require.Nil(t, err)

	// Construct a fake Kafka message using the Kafka schema wire format
	fakeKafkaMessage := []byte{0, 0, 0, 0, 77}
	fakeKafkaMessage = append(fakeKafkaMessage, avroBytes...)

	return config, mockClient, avroSchema, fakeKafkaMessage, avroMessage
}

// Test that an Avro message is retrieved from schema registry,
// decoded, unmarshaled, and the schema is saved in cache
func TestUnmarshalMessage(t *testing.T) {
	mockSchemaRegistry, mockClient, schema, kafkaMessage, avroMessage := setupMockSchemaRegistry(t)
	mockClient.On("getSchema", 77, mock.Anything).Return(schema, nil)
	mockClient.On("unmarshalKafkaMessageMap", avroMessage).Return([]error{})
	errs := mockSchemaRegistry.unmarshalMessage(context.Background(), kafkaMessage, nil)
	assert.Empty(t, errs, "there should be no errors unmarshaling")
	cachedSchema, schemaInCache := mockSchemaRegistry.schemas.Load(uint32(77))
	require.True(t, schemaInCache, "schema should be in cache")
	assert.Equal(t, schema, cachedSchema.(string))
}

// Test that an error is returned when schema registry has an error
func TestUnmarshalMessage_InvalidSchema(t *testing.T) {
	mockSchemaRegistry, mockClient, _, kafkaMessage, _ := setupMockSchemaRegistry(t)
	mockClient.On("getSchema", 77, mock.Anything).Return("", fmt.Errorf("some error"))
	errs := mockSchemaRegistry.unmarshalMessage(context.Background(), kafkaMessage, nil)
	assert.Len(t, errs, 1)
	_, schemaInCache := mockSchemaRegistry.schemas.Load(uint32(77))
	assert.Contains(t, errs[0].Error(), "some error")
	assert.False(t, schemaInCache, "schema should not be in cache")
}

// Test that an error is returned when there is a problem unmarshaling the Avro
func TestUnmarshalMessage_InvalidAvro(t *testing.T) {
	mockSchemaRegistry, mockClient, schema, kafkaMessage, avroMessage := setupMockSchemaRegistry(t)
	mockClient.On("getSchema", 77, mock.Anything).Return(schema, nil)
	mockClient.On("unmarshalKafkaMessageMap", avroMessage).Return([]error{fmt.Errorf("some error")})
	errs := mockSchemaRegistry.unmarshalMessage(context.Background(), kafkaMessage, nil)
	assert.Len(t, errs, 1)
	_, schemaInCache := mockSchemaRegistry.schemas.Load(uint32(77))
	assert.Contains(t, errs[0].Error(), "some error")
	assert.True(t, schemaInCache, "schema should be in cache")
}
