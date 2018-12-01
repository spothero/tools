package core

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
