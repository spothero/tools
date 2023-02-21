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
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/Shopify/sarama"
)

// MessageUnmarshaller are helpers for unmarshalling Kafka messages into Go types
type MessageUnmarshaller interface {
	// Unmarshal takes the contents of the ConsumerMessage and unmarshals it into the target, returning
	// any and all errors that occur during unmarshalling
	Unmarshal(ctx context.Context, msg *sarama.ConsumerMessage, target interface{}) []error
}

// Given a reflected interface value, recursively identify all fields and their related kafka tag
func extractFieldsTags(value reflect.Value) ([]reflect.Value, []string) {
	fields := make([]reflect.Value, 0)
	tags := make([]string, 0)
	for i := 0; i < value.NumField(); i++ {
		if value.Field(i).Type().Kind() == reflect.Struct && value.Field(i).Type().String() != "time.Time" {
			// hoist embedded struct tags
			newFields, newTags := extractFieldsTags(value.Field(i))
			fields = append(fields, newFields...)
			tags = append(tags, newTags...)
			continue
		}
		fields = append(fields, value.Field(i))
		tags = append(tags, value.Type().Field(i).Tag.Get("kafka"))
	}
	return fields, tags
}

// Unmarshals Avro or JSON into a struct type taking into account Kafka Connect (specifically Kafka Connect JDBC's)
// quirks. If a field from the source DBMS is nullable, Kafka connect seems
// to place the value of that field in a nested map, so we have to look for
// these maps when unmarshaling. If Kafka Connect is producing JSON, it seems to
// make every number a float64.
// Note: This function can currently handle all types of ints, bools, strings,
// and time.Time types.
func unmarshalConnectMessageMap(messageMap map[string]interface{}, target interface{}) []error {
	fields, tags := extractFieldsTags(reflect.ValueOf(target).Elem())
	errs := make([]error, 0)
	for i := 0; i < len(fields); i++ {
		tag := tags[i]
		kafkaValue, valueInMap := messageMap[tag]
		if !valueInMap {
			continue
		}
		field := fields[i]

		// handle Kafka Connect placing nullable values as nested
		// map[string]interface{} where the (single) key of the map is the type
		// by moving the actual value out of the nested map
		// ex: {"nullable_int": {"int": 0}, "nullable_string": {"string: "abc"}}
		//  -> {"nullable_int": 0, "nullable_string": "abc"}
		if v, ok := kafkaValue.(map[string]interface{}); ok {
			kafkaValue = v[reflect.ValueOf(v).MapKeys()[0].String()]
		}

		if kafkaValue == nil {
			continue
		}
		var err error
		if field.CanSet() && field.IsValid() {
			fieldKind := field.Type().Kind()
			switch fieldKind {
			case reflect.Bool:
				// Booleans come through from Kafka Connect as int32, int64, or actual bools
				if b, ok := kafkaValue.(int32); ok {
					field.SetBool(b > 0)
				} else if b, ok := kafkaValue.(int64); ok {
					field.SetBool(b > 0)
				} else if b, ok := kafkaValue.(float64); ok {
					field.SetBool(b > 0)
				} else if b, ok := kafkaValue.(bool); ok {
					field.SetBool(b)
				} else {
					err = fmt.Errorf("couldn't set bool field with tag %s", tag)
				}
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				// Avro only has int32 and int64 values so we just need to check those
				if i, ok := kafkaValue.(int32); ok {
					field.SetInt(int64(i))
				} else if i, ok := kafkaValue.(int64); ok {
					field.SetInt(i)
				} else if i, ok := kafkaValue.(float64); ok {
					field.SetInt(int64(i))
				} else {
					err = fmt.Errorf("couldn't set int field with tag %s", tag)
				}
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				if i, ok := kafkaValue.(int32); ok {
					field.SetUint(uint64(i))
				} else if i, ok := kafkaValue.(int64); ok {
					field.SetUint(uint64(i))
				} else if i, ok := kafkaValue.(float64); ok {
					field.SetUint(uint64(i))
				} else {
					err = fmt.Errorf("couldn't set uint field with tag %s", tag)
				}
			case reflect.Float32, reflect.Float64:
				if i, ok := kafkaValue.(float32); ok {
					field.SetFloat(float64(i))
				} else if i, ok := kafkaValue.(float64); ok {
					field.SetFloat(i)
				} else {
					err = fmt.Errorf("couldn't set float field with tag %s", tag)
				}
			case reflect.String:
				if s, ok := kafkaValue.(string); ok {
					field.SetString(s)
				} else {
					err = fmt.Errorf("couldn't set string field with tag %s", tag)
				}
			case reflect.Struct:
				if field.Type().String() == "time.Time" {
					// times are encoded as int64 milliseconds in Avro
					if t, ok := kafkaValue.(int64); ok {
						timeVal := time.Unix(0, t*1000000)
						field.Set(reflect.ValueOf(timeVal))
					} else if t, ok := kafkaValue.(float64); ok {
						timeVal := time.Unix(0, int64(t)*1000000)
						field.Set(reflect.ValueOf(timeVal))
					} else if t, ok := kafkaValue.(string); ok {
						// try decoding as RFC3339 time string
						timeVal, parseErr := time.Parse(time.RFC3339, t)
						if parseErr == nil {
							field.Set(reflect.ValueOf(timeVal))
						} else {
							err = fmt.Errorf("failed to parse time field with tag %s, reason: %w", tag, parseErr)
						}
					} else {
						err = fmt.Errorf("couldn't set time field with tag %s", tag)
					}
					break
				}
				fallthrough
			default:
				err = fmt.Errorf(
					"unhandled type %s, field with tag %s will not be set", field.Type().String(), tag)
			}
		} else {
			err = fmt.Errorf("cannot set invalid field with tag %s", tag)
		}
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// ConnectAvroUnmarshaller is a helper for unmarshalling Kafka Avro messages, taking into account the quirks
// of the Kafka Connect producer's format
type ConnectAvroUnmarshaller struct {
	SchemaRegistryClient
}

// Unmarshal takes the contents of the ConsumerMessage and unmarshals it into the target using avro decoding, returning
// any and all errors that occur during unmarshalling
func (u ConnectAvroUnmarshaller) Unmarshal(ctx context.Context, msg *sarama.ConsumerMessage, target interface{}) []error {
	messageData, err := u.DecodeKafkaAvroMessage(ctx, msg)
	if err != nil {
		return []error{err}
	}
	messageMap, ok := messageData.(map[string]interface{})
	if !ok {
		return []error{fmt.Errorf("failed to unmarshal Kafka message becasue the data is not a map")}
	}
	return unmarshalConnectMessageMap(messageMap, target)
}

// ConnectJSONUnmarshaller is a helper for unmarshalling Kafka JSON messages, taking into account the quirks
// of the Kafka Connect producer's format
type ConnectJSONUnmarshaller struct{}

// Unmarshal takes the contents of the ConsumerMessage and unmarshals it into the target using JSON decoding, returning
// any and all errors that occur during unmarshalling
func (u ConnectJSONUnmarshaller) Unmarshal(ctx context.Context, msg *sarama.ConsumerMessage, target interface{}) []error {
	message := make(map[string]interface{})
	if err := json.Unmarshal(msg.Value, &message); err != nil {
		return []error{fmt.Errorf("failed to unmarshal Kafka message because the JSON was invalid: %w", err)}
	}
	return unmarshalConnectMessageMap(message, target)
}
