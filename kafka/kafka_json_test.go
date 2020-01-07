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

func (mju *mockJSONUnmarshaler) unmarshalMessageMap(kafkaMessageMap map[string]interface{}, target interface{}) []error {
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
	mju.On("unmarshalMessageMap", expectedPayload).Return([]error{})
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
	mju.On("unmarshalMessageMap", expectedPayload).Return([]error{fmt.Errorf("calcium levels too low to go to flavortown")})
	err := jmu.UnmarshalMessage(context.Background(), kafkaMessage, mju)
	assert.NotNil(t, err)
}
