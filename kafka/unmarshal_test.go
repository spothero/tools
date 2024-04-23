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
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalConnectMessageMap(t *testing.T) {
	tests := []struct {
		setup  func() map[string]interface{}
		verify func(t *testing.T, msg map[string]interface{})
		name   string
	}{
		{
			name: "all supported types are correctly handled",
			setup: func() map[string]interface{} {
				// Build a dummy decoded Kafka message
				kafkaConnectMessage := make(map[string]interface{})
				kafkaConnectMessage["a"] = int32(1)
				kafkaConnectMessage["b"] = int32(2)
				kafkaConnectMessage["c"] = int32(3)
				kafkaConnectMessage["d"] = int32(4)
				kafkaConnectMessage["e"] = int32(5)
				kafkaConnectMessage["f"] = int32(6)
				kafkaConnectMessage["g"] = int32(7)
				kafkaConnectMessage["h"] = int32(8)
				kafkaConnectMessage["i"] = int32(9)
				kafkaConnectMessage["j"] = int32(10)
				kafkaConnectMessage["k"] = int32(0)
				kafkaConnectMessage["l"] = "abc"
				kafkaConnectMessage["m"] = time.Unix(1522083600, 0).UTC().Unix() * 1000
				kafkaConnectMessage["n"] = float32(1.1)
				kafkaConnectMessage["o"] = float64(2.2)
				kafkaConnectMessage["p"] = true
				kafkaConnectMessage["q"] = "2018-08-23T15:56:00-05:00"
				return kafkaConnectMessage
			},
			verify: func(t *testing.T, msg map[string]interface{}) {
				// Define a struct containing every supported type
				type unmarshalTarget struct {
					Q time.Time `kafka:"q"`
					M time.Time `kafka:"m"`
					L string    `kafka:"l"`
					E int64     `kafka:"e"`
					F uint      `kafka:"f"`
					O float64   `kafka:"o"`
					A int       `kafka:"a"`
					J uint64    `kafka:"j"`
					D int32     `kafka:"d"`
					I uint32    `kafka:"i"`
					N float32   `kafka:"n"`
					H uint16    `kafka:"h"`
					C int16     `kafka:"c"`
					K bool      `kafka:"k"`
					G uint8     `kafka:"g"`
					P bool      `kafka:"p"`
					B int8      `kafka:"b"`
				}
				target := &unmarshalTarget{}
				errs := unmarshalConnectMessageMap(msg, target)
				assert.Empty(t, errs)
				assert.Equal(t, 1, target.A)
				assert.Equal(t, int8(2), target.B)
				assert.Equal(t, int16(3), target.C)
				assert.Equal(t, int32(4), target.D)
				assert.Equal(t, int64(5), target.E)
				assert.Equal(t, uint(6), target.F)
				assert.Equal(t, uint8(7), target.G)
				assert.Equal(t, uint16(8), target.H)
				assert.Equal(t, uint32(9), target.I)
				assert.Equal(t, uint64(10), target.J)
				assert.False(t, target.K)
				assert.Equal(t, "abc", target.L)
				assert.Equal(t, time.Unix(1522083600, 0), target.M)
				assert.Equal(t, float32(1.1), target.N)
				assert.Equal(t, float64(2.2), target.O)
				assert.Equal(t, true, target.P)
				central, err := time.LoadLocation("America/Chicago")
				require.NoError(t, err)
				assert.Equal(t, time.Date(2018, 8, 23, 15, 56, 0, 0, central).UTC(), target.Q.UTC())
			},
		}, {
			name: "nullable fields are handled correctly",
			setup: func() map[string]interface{} {
				message := make(map[string]interface{})
				nullable := make(map[string]interface{})
				nullable["int"] = int32(123)
				message["a"] = nullable
				return message
			},
			verify: func(t *testing.T, msg map[string]interface{}) {
				type unmarshalTarget struct {
					A int `kafka:"a"`
				}
				target := &unmarshalTarget{}
				errs := unmarshalConnectMessageMap(msg, target)
				assert.Empty(t, errs)
				assert.Equal(t, 123, target.A)
			},
		}, {
			name: "unset fields are ummarshalled correctly",
			setup: func() map[string]interface{} {
				return make(map[string]interface{})
			},
			verify: func(t *testing.T, msg map[string]interface{}) {
				type unmarshalTarget struct {
					A int `kafka:"a"`
				}
				target := &unmarshalTarget{}
				errs := unmarshalConnectMessageMap(msg, target)
				assert.Empty(t, errs)
				assert.Equal(t, 0, target.A)
			},
		}, {
			name: "unsupported types return errors",
			setup: func() map[string]interface{} {
				message := make(map[string]interface{})
				message["a"] = []byte{'T', 'H', 'A', 'N', 'K'}
				return message
			},
			verify: func(t *testing.T, msg map[string]interface{}) {
				type unmarshalTarget struct {
					A []byte `kafka:"a"`
				}
				target := &unmarshalTarget{}
				errs := unmarshalConnectMessageMap(msg, target)
				require.Len(t, errs, 1)
				expectedErr := fmt.Errorf("unhandled type []uint8, field with tag a will not be set")
				assert.Equal(t, expectedErr, errs[0])
			},
		}, {
			name: "unexported fields return errors",
			setup: func() map[string]interface{} {
				message := make(map[string]interface{})
				message["a"] = 1
				return message
			},
			verify: func(t *testing.T, msg map[string]interface{}) {
				type unmarshalTarget struct {
					_ int `kafka:"a"`
				}
				target := &unmarshalTarget{}
				errs := unmarshalConnectMessageMap(msg, target)
				require.Len(t, errs, 1)
				expectedErr := fmt.Errorf("cannot set invalid field with tag a")
				assert.Equal(t, expectedErr, errs[0])
			},
		}, {
			name: "embedded structs are unmarshalled correctly",
			setup: func() map[string]interface{} {
				message := make(map[string]interface{})
				message["a"] = int32(1)
				message["b"] = int32(2)
				message["c"] = "data"
				message["d"] = time.Unix(1522083600, 0).UTC().Unix() * 1000
				return message
			},
			verify: func(t *testing.T, msg map[string]interface{}) {
				type DoubleNestedTarget struct {
					D time.Time `kafka:"d"`
					C string    `kafka:"c"`
				}
				type NestedTarget struct {
					DoubleNestedTarget
					B int `kafka:"b"`
				}
				type unmarshalTarget struct {
					NestedTarget
					A int `kafka:"a"`
				}
				target := &unmarshalTarget{}
				errs := unmarshalConnectMessageMap(msg, target)
				assert.Empty(t, errs)
				assert.Equal(t, 1, target.A)
				assert.Equal(t, 2, target.B)
				assert.Equal(t, "data", target.C)
				assert.Equal(t, time.Unix(1522083600, 0), target.D)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.verify(t, test.setup())
		})
	}
}

func TestExtractFieldsTags(t *testing.T) {
	type DoubleNestedTarget struct {
		A time.Time `kafka:"a"`
		B string    `kafka:"b"`
	}
	type NestedTarget struct {
		DoubleNestedTarget
		C int `kafka:"c"`
	}
	type unmarshalTarget struct {
		NestedTarget
		D int `kafka:"d"`
	}
	target := &unmarshalTarget{}
	reflected := reflect.ValueOf(target).Elem()
	fields, tags := extractFieldsTags(reflect.ValueOf(target).Elem())
	assert.Equal(t, fields, []reflect.Value{
		reflected.FieldByName("A"),
		reflected.FieldByName("B"),
		reflected.FieldByName("C"),
		reflected.FieldByName("D"),
	})
	assert.Equal(t, tags, []string{"a", "b", "c", "d"})
}

type dummyMsg struct {
	Name string `kafka:"name"`
}

func TestConnectAvroUnmarshaller_Unmarshal(t *testing.T) {
	tests := []struct {
		msg             *sarama.ConsumerMessage
		name            string
		expectedOutcome dummyMsg
		avroDecodeErr   bool
		expectErr       bool
	}{
		{
			name:            "connect message is unmarshalled from avro",
			msg:             newAvroMessage(t),
			expectedOutcome: dummyMsg{Name: "Guy Fieri"},
		}, {
			name:          "error decoding avro message returns error",
			msg:           newAvroMessage(t),
			avroDecodeErr: true,
			expectErr:     true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := SchemaRegistryClient{
				SchemaRegistryConfig: SchemaRegistryConfig{URL: "schema.registry"},
				cache:                &sync.Map{},
				client:               http.Client{Transport: &mockTransport{t: t, expectedRequest: expectedRequestEmpty()}},
			}
			getSchema := client.client.Transport.(*mockTransport).On("RoundTrip", mock.Anything)
			if test.avroDecodeErr {
				getSchema.Return(&http.Response{StatusCode: http.StatusNotFound}, nil)
			} else {
				getSchema.Return(&http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(
						strings.NewReader(
							fmt.Sprintf("{\"schema\": \"%s\"}", strings.Replace(avroSchema, "\"", "\\\"", -1)))),
				}, nil)
			}
			target := dummyMsg{}
			errs := ConnectAvroUnmarshaller{client}.Unmarshal(context.Background(), test.msg, &target)
			if test.expectErr {
				assert.GreaterOrEqual(t, len(errs), 1)
				return
			}
			assert.Equal(t, test.expectedOutcome, target)
		})
	}
}

func TestConnectJSONUnmarshaller_Unmarshal(t *testing.T) {
	tests := []struct {
		name            string
		msg             *sarama.ConsumerMessage
		expectedOutcome dummyMsg
		expectErr       bool
	}{
		{
			"connect message is unmarshalled from json",
			&sarama.ConsumerMessage{Value: sarama.ByteEncoder("{\"name\": \"JSON Derulo\"}")},
			dummyMsg{Name: "JSON Derulo"},
			false,
		}, {
			"error decoding json message returns error",
			&sarama.ConsumerMessage{Value: sarama.ByteEncoder("{\"name\":")},
			dummyMsg{},
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			target := dummyMsg{}
			errs := ConnectJSONUnmarshaller{}.Unmarshal(context.Background(), test.msg, &target)
			if test.expectErr {
				assert.GreaterOrEqual(t, len(errs), 1)
				return
			}
			assert.Equal(t, test.expectedOutcome, target)
		})
	}
}
