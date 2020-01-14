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

package mock

// CircuitError implements the circuit.Error interface for tests
type CircuitError struct {
	Message             string
	ConcurrencyExceeded bool
	CircuitOpened       bool
}

// Error returns the error message for Circuit Error structs
func (mce CircuitError) Error() string {
	return mce.Message
}

// ConcurrencyLimitReached returns whether the concurrency limit has been reached for
// Circuit Error structs
func (mce CircuitError) ConcurrencyLimitReached() bool {
	return mce.ConcurrencyExceeded
}

// CircuitOpen returns whether the circuit is open for Circuit Error structs
func (mce CircuitError) CircuitOpen() bool {
	return mce.CircuitOpened
}
