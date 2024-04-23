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

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/mock"
)

// MockSchemaRegistryClient defines an interface for mocking all interfaces that are satisfied by schema registry client.
type MockSchemaRegistryClient struct {
	mock.Mock
}

// CreateSchema implements the SchemaRegistryProducer interface.
func (mc *MockSchemaRegistryClient) CreateSchema(ctx context.Context, subject string, schema string, isKey bool) (*SchemaResponse, error) {
	args := mc.Called(ctx, subject, schema, isKey)
	return args.Get(0).(*SchemaResponse), args.Error(1)
}

// EncodeKafkaAvroMessage implements the SchemaRegistryProducer interface.
func (mc *MockSchemaRegistryClient) EncodeKafkaAvroMessage(ctx context.Context, schemaID uint, message interface{}) ([]byte, error) {
	args := mc.Called(ctx, schemaID, message)
	return args.Get(0).([]byte), args.Error(1)
}

// GetSchema implements the SchemaRegistryConsumer interface.
func (mc *MockSchemaRegistryClient) GetSchema(ctx context.Context, id uint) (string, error) {
	args := mc.Called(ctx, id)
	return args.String(0), args.Error(1)
}

// CheckSchema implements the SchemaRegistryConsumer interface.
func (mc *MockSchemaRegistryClient) CheckSchema(ctx context.Context, subject string, schema string, isKey bool) (*SchemaResponse, error) {
	args := mc.Called(ctx, subject, schema, isKey)
	return args.Get(0).(*SchemaResponse), args.Error(1)
}

// DecodeKafkaAvroMessage implements the SchemaRegistryConsumer interface.
func (mc *MockSchemaRegistryClient) DecodeKafkaAvroMessage(ctx context.Context, message *sarama.ConsumerMessage) (interface{}, error) {
	args := mc.Called(ctx, message)
	return args.Get(0), args.Error(1)
}
