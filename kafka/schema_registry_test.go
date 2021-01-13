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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
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

const avroSchema = `{"type": "record", "name": "test", "fields": [{"name": "name", "type": "string"}]}`
const schemaID = uint(77)
const schema = "it's a schema"

type expectedRequest struct {
	url         string
	method      string
	requestBody string
}

type mockTransport struct {
	mock.Mock
	t *testing.T
	expectedRequest expectedRequest
}

func readerToString(reader io.Reader) string {
	buf := new(strings.Builder)
	_, _ = io.Copy(buf, reader)
	return buf.String()
}

func buildSchemaRegistryServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		responsePayload := schemaResponse{
			Subject: "test-subject",
			Version: 1,
			Schema:  "test schema",
			ID:      77,
		}
		response, _ := json.Marshal(responsePayload)

		switch req.URL.String() {
		case "/schemas/ids/77":
			assert.Equal(t, http.NoBody, req.Body)
			_, _ = rw.Write(response)
		case "/schemas/ids/100":
			resp := errorResponse{
				ErrorCode: 40403,
				Message: "Schema not found",
			}
			jsonResp, _ := json.Marshal(resp)
			rw.WriteHeader(http.StatusNotFound)
			_, _ = rw.Write(jsonResp)
			assert.Equal(t, http.NoBody, req.Body)
		case "/schemas/ids/1000":
			rw.WriteHeader(http.StatusTeapot)
			assert.Equal(t, http.NoBody, req.Body)
		case "/schemas/ids/999":
			_, _ = rw.Write([]byte("not json"))
			assert.Equal(t, http.NoBody, req.Body)
		case "/subjects/test-subject-value":
			_, _ = rw.Write(response)
			assert.Equal(t, "{\"schema\":\"test schema\"}", readerToString(req.Body))
		case "/subjects/test-subject-not-found-value":
			resp := errorResponse{
				ErrorCode: 40401,
				Message: "Subject not found.",
			}
			jsonResp, _ := json.Marshal(resp)
			rw.WriteHeader(http.StatusNotFound)
			_, _ = rw.Write(jsonResp)
			assert.Equal(t, "{\"schema\":\"test schema\"}", readerToString(req.Body))
		case "/subjects/test-subject-not-json-value":
			_, _ = rw.Write([]byte("not json"))
			assert.Equal(t, "{\"schema\":\"test schema\"}", readerToString(req.Body))
		case "/subjects/test-subject-unexpected-response-value":
			rw.WriteHeader(http.StatusTeapot)
			assert.Equal(t, "{\"schema\":\"test schema\"}", readerToString(req.Body))
		case "/subjects/test-subject-value/versions":
			_, _ = rw.Write(response)
			assert.Equal(t, "{\"schema\":\"test schema\"}", readerToString(req.Body))
		case "/subjects/test-subject-incompatible-value/versions":
			rw.WriteHeader(http.StatusConflict)
			assert.Equal(t, "{\"schema\":\"test schema\"}", readerToString(req.Body))
		case "/subjects/test-subject-unprocessable-value/versions":
			resp := errorResponse{
				ErrorCode: 42201,
				Message: "Input schema is an invalid Avro schema",
			}
			jsonResp, _ := json.Marshal(resp)
			rw.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = rw.Write(jsonResp)
			assert.Equal(t, "{\"schema\":\"test schema\"}", readerToString(req.Body))
		case "/subjects/test-subject-not-json-value/versions":
			_, _ = rw.Write([]byte("not json"))
			assert.Equal(t, "{\"schema\":\"test schema\"}", readerToString(req.Body))
		case "/subjects/test-subject-unexpected-response-value/versions":
			rw.WriteHeader(http.StatusTeapot)
			assert.Equal(t, "{\"schema\":\"test schema\"}", readerToString(req.Body))
		default:
			assert.Error(t, errors.New("unhandled request"))
		}

	}))
}


func TestSchemaRegistryClient_GetSchema(t *testing.T) {

	tests := []struct {
		name              string
		schemaID          uint
		schema            string
		error             string
		url               string
	}{
		{
			name: "schema is retrieved from schema registry",
			schemaID: 77,
			schema: "test schema",
		},
		{
			name: "404 returns error",
			schemaID: 100,
			error: "schema 100 not found",
		}, {
			name: "non-200 returns error",
			schemaID: 1000,
			error: "error while retrieving schema; schema registry returned unhandled status code 418",
		},
		{
			name: "bad json returns error",
			schemaID: 999,
			error: "invalid character 'o' in literal null (expecting 'u')",
		},
		{
			name: "invalid url returns error",
			schemaID: 77,
			error: "failed to build schema registry http request: parse \"💀:///schemas/ids/77\": first path segment in URL cannot contain colon",
			url: "💀://",
		},
	}

	server := buildSchemaRegistryServer(t)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			url := server.URL
			if test.url != "" {
				url = test.url
			}

			client := SchemaRegistryClient{
				SchemaRegistryConfig: SchemaRegistryConfig{URL: url},
				client:               http.Client{},
				cache:                sync.Map{},
			}

			schema, err := client.GetSchema(context.Background(), test.schemaID)

			if test.error == "" {
				assert.NoError(t, err)
			} else {
				assert.NotNil(t, err)
				assert.Equal(t, test.error, err.Error())
			}
			assert.Equal(t, test.schema, schema)
		})
	}
}

func TestSchemaRegistryClient_CheckSchema(t *testing.T) {

	tests := []struct {
		name              string
		subject           string
		schema            string
		error             string
		url               string
		schemaResponse    *schemaResponse
	}{
		{
			name: "schema is found in the schema registry",
			subject: "test-subject",
			schema: "test schema",
			schemaResponse: &schemaResponse{
				"test-subject",
				1,
				"test schema",
				77,
			},
		},
		{
			name: "subject is not found in the schema registry",
			subject: "test-subject-not-found",
			schema: "test schema",
			error: "Subject not found., error code 40401",
		},
		{
			name: "schema registry returns bad json",
			subject: "test-subject-not-json",
			schema: "test schema",
			error: "invalid character 'o' in literal null (expecting 'u')",
		},
		{
			name: "non-200 returns error",
			subject: "test-subject-unexpected-response",
			schema: "test schema",
			error: "error while checking schema; schema registry returned unhandled status code 418",
		},
		{
			name: "invalid url returns error",
			subject: "test-subject",
			schema: "test schema",
			error: "failed to build schema registry http request: parse \"💀:///subjects/test-subject-value\": first path segment in URL cannot contain colon",
			url: "💀://",
		},
	}

	server := buildSchemaRegistryServer(t)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			url := server.URL
			if test.url != "" {
				url = test.url
			}

			client := SchemaRegistryClient{
				SchemaRegistryConfig: SchemaRegistryConfig{URL: url},
				client:               http.Client{},
				cache:                sync.Map{},
			}

			schemaResponse, err := client.CheckSchema(context.Background(), test.subject, test.schema, false)

			if test.error == "" {
				assert.NoError(t, err)
			} else {
				assert.NotNil(t, err)
				assert.Equal(t, test.error, err.Error())
			}
			assert.Equal(t, test.schemaResponse, schemaResponse)
		})
	}
}

func TestSchemaRegistryClient_CreateSchema(t *testing.T) {

	tests := []struct {
		name              string
		subject           string
		schema            string
		error             string
		url               string
		schemaResponse    *schemaResponse
	}{
		{
			name: "schema is created in the schema registry",
			subject: "test-subject",
			schema: "test schema",
			schemaResponse: &schemaResponse{
				"test-subject",
				1,
				"test schema",
				77,
			},
		},
		{
			name: "schema is not created due to incompatibility",
			subject: "test-subject-incompatible",
			schema: "test schema",
			error: "incompatible schema",
		},
		{
			name: "schema is not created due to incompatibility",
			subject: "test-subject-unprocessable",
			schema: "test schema",
			error: "Input schema is an invalid Avro schema, error code 42201",
		},
		{
			name: "schema registry returns bad json",
			subject: "test-subject-not-json",
			schema: "test schema",
			error: "invalid character 'o' in literal null (expecting 'u')",
		},
		{
			name: "non-200 returns error",
			subject: "test-subject-unexpected-response",
			schema: "test schema",
			error: "error while creating schema; schema registry returned unhandled status code 418",
		},
		{
			name: "invalid url returns error",
			subject: "test-subject",
			schema: "test schema",
			error: "parse \"💀:///subjects/test-subject-value/versions\": first path segment in URL cannot contain colon",
			url: "💀://",
		},
	}

	server := buildSchemaRegistryServer(t)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			url := server.URL
			if test.url != "" {
				url = test.url
			}

			client := SchemaRegistryClient{
				SchemaRegistryConfig: SchemaRegistryConfig{URL: url},
				client:               http.Client{},
				cache:                sync.Map{},
			}

			schemaResponse, err := client.CreateSchema(context.Background(), test.subject, test.schema, false)

			if test.error == "" {
				assert.NoError(t, err)
			} else {
				assert.NotNil(t, err)
				assert.Equal(t, test.error, err.Error())
			}
			assert.Equal(t, test.schemaResponse, schemaResponse)
		})
	}
}

func TestSchemaRegistryConfig_RegisterFlags(t *testing.T) {
	flags := pflag.NewFlagSet("pflags", pflag.PanicOnError)
	c := SchemaRegistryConfig{}
	c.RegisterFlags(flags)
	err := flags.Parse([]string{"--kafka-schema-registry-url", "http://schema.registry"})
	require.NoError(t, err)
	assert.Equal(t, "http://schema.registry", c.URL)
}

func expectedRequestEmpty() expectedRequest {
	return expectedRequest{
		"schema.registry/schemas/ids/77",
		http.MethodGet,
		"",
	}
}

func (m *mockTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	call := m.Called(request)
	m.t.Helper()
	assert.Equal(m.t, "application/vnd.schemaregistry.v1+json", request.Header.Get("Accept"))
	assert.Equal(m.t, m.expectedRequest.url, request.URL.String())
	assert.Equal(m.t, m.expectedRequest.method, request.Method)
	if m.expectedRequest.requestBody == "" {
		assert.Nil(m.t, request.Body)
	} else {
		assert.Equal(m.t, m.expectedRequest.requestBody, readerToString(request.Body))
	}
	return call.Get(0).(*http.Response), call.Error(1)
}

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
				cache:                sync.Map{},
				client:               http.Client{Transport: &mockTransport{t: t, expectedRequest: expectedRequestEmpty()}},
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
