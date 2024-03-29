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

package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestNewDefaultClientConfig(t *testing.T) {
	cc := NewDefaultClientConfig(context.Background())
	assert.Equal(t, "127.0.0.1", cc.Address)
	assert.Equal(t, uint16(9111), cc.Port)
	assert.NotNil(t, cc.UnaryInterceptors)
	assert.NotNil(t, cc.StreamInterceptors)
	assert.NotNil(t, cc.Options)
}

func TestNewDefaultTLSClientConfig(t *testing.T) {
	cc := NewDefaultTLSClientConfig(context.Background())
	assert.Equal(t, "127.0.0.1", cc.Address)
	assert.Equal(t, uint16(9111), cc.Port)
	assert.NotNil(t, cc.UnaryInterceptors)
	assert.NotNil(t, cc.StreamInterceptors)
	assert.NotNil(t, cc.Options)
}

func TestGetConn(t *testing.T) {
	conn, err := ClientConfig{
		PropagateAuthHeaders: true,
		RetryServerErrors:    true,
		Options:              []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())},
	}.GetConn()
	assert.NoError(t, err)
	assert.NotNil(t, conn)
	_ = conn.Close()
}
