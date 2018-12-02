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

package core

import (
	"fmt"
	"reflect"
	"time"

	"go.uber.org/zap"
)

type kafkaMessageUnmarshaler interface {
	unmarshalKafkaMessageMap(kafkaMessageMap map[string]interface{}, target interface{}) []error
}
type kafkaMessageDecoder struct{}

// Unmarshal Avro or JSON into a struct type taking into account Kafka Connect's
// quirks. If a field from the source DBMS is nullable, Kafka connect seems
// to place the value of that field in a nested map, so we have to look for
// these maps when unmarshaling. If Kafka Connect is producing JSON, it seems to
// make every number a float64.
// Note: This function can currently handle all types of ints, bools, strings,
// and time.Time types.
func (kmd *kafkaMessageDecoder) unmarshalKafkaMessageMap(kafkaMessageMap map[string]interface{}, target interface{}) []error {
	valueOfStructure := reflect.ValueOf(target).Elem()
	typeOfStructure := valueOfStructure.Type()
	errs := make([]error, 0)
	for i := 0; i < valueOfStructure.NumField(); i++ {
		tag := typeOfStructure.Field(i).Tag.Get("kafka")
		kafkaValue, valueInMap := kafkaMessageMap[tag]
		if !valueInMap {
			continue
		}
		field := valueOfStructure.Field(i)

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
			fieldType := field.Type().String()
			switch fieldType {
			case "bool":
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
					err = fmt.Errorf("error unmarshaling Kafka message, couldn't set bool field with tag %s", tag)
				}
			case "int", "int8", "int16", "int32", "int64":
				// Avro only has int32 and int64 values so we just need to check those
				if i, ok := kafkaValue.(int32); ok {
					field.SetInt(int64(i))
				} else if i, ok := kafkaValue.(int64); ok {
					field.SetInt(i)
				} else if i, ok := kafkaValue.(float64); ok {
					field.SetInt(int64(i))
				} else {
					err = fmt.Errorf("error unmarshaling Kafka message, couldn't set int field with tag %s", tag)
				}
			case "uint", "uint8", "uint16", "uint32", "uint64":
				if i, ok := kafkaValue.(int32); ok {
					field.SetUint(uint64(i))
				} else if i, ok := kafkaValue.(int64); ok {
					field.SetUint(uint64(i))
				} else if i, ok := kafkaValue.(float64); ok {
					field.SetUint(uint64(i))
				} else {
					err = fmt.Errorf("error unmarshaling Kafka message, couldn't set uint field with tag %s", tag)
				}
			case "float32", "float64":
				if i, ok := kafkaValue.(float32); ok {
					field.SetFloat(float64(i))
				} else if i, ok := kafkaValue.(float64); ok {
					field.SetFloat(float64(i))
				} else {
					err = fmt.Errorf("error unmarshaling Kafka message, couldn't set float field with tag %s", tag)
				}
			case "string":
				if s, ok := kafkaValue.(string); ok {
					field.SetString(s)
				} else {
					err = fmt.Errorf("error unmarshaling Kafka message, couldn't set string field with tag %s", tag)
				}
			case "time.Time":
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
						err = fmt.Errorf("error unmarshaling Kafka message, failed to parse time field with tag %s, reason: %s", tag, parseErr.Error())
					}
				} else {
					err = fmt.Errorf("error unmarshaling Kafka message, couldn't set time field with tag %s", tag)
				}
			default:
				err = fmt.Errorf(
					"unhandled Avro type %s, field with tag %s will not be set", field.Type().String(), tag)
				Logger.Error(
					"Unhandled Avro type! This field will not be set!",
					zap.String("field_type", field.Type().String()), zap.String("field_tag", tag))
			}
		} else {
			err = fmt.Errorf("cannot set invalid field with tag %s", tag)
			Logger.Error(
				"Cannot set invalid field", zap.String("field_tag", tag),
				zap.Bool("field_can_set", field.CanSet()),
				zap.Bool("field_is_valid", field.IsValid()))
		}
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}
