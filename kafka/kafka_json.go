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
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/Shopify/sarama"
	"github.com/opentracing/opentracing-go"
	"go.uber.org/zap"
)

type jsonMessageUnmarshaler struct {
	messageUnmarshaler kafkaMessageUnmarshaler
}

// Implements the KafkaMessageUnmarshaler interface and decodes
// Kafka messages from JSON
func (jmu *jsonMessageUnmarshaler) UnmarshalMessage(
	ctx context.Context,
	msg *sarama.ConsumerMessage,
	target interface{},
) error {
	span, _ := opentracing.StartSpanFromContext(ctx, "unmarshal-kafka-json")
	defer span.Finish()
	message := make(map[string]interface{})
	if err := json.Unmarshal(msg.Value, &message); err != nil {
		return err
	}
	unmarshalErrs := jmu.messageUnmarshaler.unmarshalKafkaMessageMap(message, target)
	if len(unmarshalErrs) > 0 {
		Logger.Error(
			"Unable to unmarshal from JSON", zap.Errors("errors", unmarshalErrs),
			zap.String("type", reflect.TypeOf(target).String()))
		return fmt.Errorf("unable to unmarshal from JSON")
	}
	return nil
}
