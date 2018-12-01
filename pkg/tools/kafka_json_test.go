package core

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockJSONUnmarshaler struct {
	mock.Mock
}

func (mju *mockJSONUnmarshaler) unmarshalKafkaMessageMap(kafkaMessageMap map[string]interface{}, target interface{}) []error {
	args := mju.Called(kafkaMessageMap)
	return args.Get(0).([]error)
}

func TestUnmarshalJsonMessage(t *testing.T) {
	mju := &mockJSONUnmarshaler{}
	jmu := jsonMessageUnmarshaler{messageUnmarshaler: mju}
	fakeJSONMessage := bytes.NewBufferString("{\"where_to_go\": \"flavortown\"}")
	kafkaMessage := &sarama.ConsumerMessage{Value: fakeJSONMessage.Bytes()}
	expectedPayload := make(map[string]interface{})
	expectedPayload["where_to_go"] = "flavortown"
	mju.On("unmarshalKafkaMessageMap", expectedPayload).Return([]error{})
	err := jmu.UnmarshalMessage(context.Background(), kafkaMessage, nil)
	assert.Nil(t, err)
}

func TestUnmarshalJsonMessage_InvalidJson(t *testing.T) {
	mju := &mockJSONUnmarshaler{}
	jmu := jsonMessageUnmarshaler{messageUnmarshaler: mju}
	fakeJSONMessage := bytes.NewBufferString("{\"where_to_go\": \"flavortown\"}}}")
	kafkaMessage := &sarama.ConsumerMessage{Value: fakeJSONMessage.Bytes()}
	err := jmu.UnmarshalMessage(context.Background(), kafkaMessage, nil)
	assert.NotNil(t, err)
}

func TestUnmarshalJsonMessage_ErrorUnmarshaling(t *testing.T) {
	mju := &mockJSONUnmarshaler{}
	jmu := jsonMessageUnmarshaler{messageUnmarshaler: mju}
	fakeJSONMessage := bytes.NewBufferString("{\"where_to_go\": \"flavortown\"}")
	kafkaMessage := &sarama.ConsumerMessage{Value: fakeJSONMessage.Bytes()}
	expectedPayload := make(map[string]interface{})
	expectedPayload["where_to_go"] = "flavortown"
	mju.On("unmarshalKafkaMessageMap", expectedPayload).Return([]error{fmt.Errorf("calcium levels too low to go to flavortown")})
	err := jmu.UnmarshalMessage(context.Background(), kafkaMessage, mju)
	assert.NotNil(t, err)
}
