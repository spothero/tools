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

package tools

import (
	"os"
)

// GetEnvVal takes in an environmental variable key (key) and a default value (defaultVal)
// and returns the value of the environmental variable key if found or the defualt value provided.
func GetEnvVal(key string, defaultVal string) string {
	val, ok := os.LookupEnv(key); if ok{
		return val
	}
	return defaultVal
}