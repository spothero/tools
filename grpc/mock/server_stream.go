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

package mock

import (
	"context"

	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/metadata"
)

// MockServerStream defines a mock GRPC server stream implementation for use in stream tests
type MockServerStream struct {
	mock.Mock
}

// SetHeader mocks the ServerStream SetHeader function
func (mss *MockServerStream) SetHeader(md metadata.MD) error {
	return mss.Called(md).Error(0)
}

// SendHeader mocks the ServerStream SendHeader function
func (mss *MockServerStream) SendHeader(md metadata.MD) error {
	return mss.Called(md).Error(0)
}

// SetTrailer mocks the ServerStream SetTrailer function
func (mss *MockServerStream) SetTrailer(md metadata.MD) {
	mss.Called(md)
}

// Context mocks the ServerStream Context function
func (mss *MockServerStream) Context() context.Context {
	mss.Called()
	return context.Background()
}

// SendMsg mocks the ServerStream SendMsg function
func (mss *MockServerStream) SendMsg(m interface{}) error {
	return mss.Called(m).Error(0)
}

// RecvMsg mocks the ServerStream RecvMsg function
func (mss *MockServerStream) RecvMsg(m interface{}) error {
	return mss.Called(m).Error(0)
}
