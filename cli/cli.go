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

package cli

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// CobraBindEnvironmentVariables can be used at the root command level of a cobra CLI hierarchy to allow
// all command-line variables to be set by environment variables as well. Note that
// skewered-variable-names will automatically be translated to skewered_variable_names
// for compatibility with environment variables.
//
// In addition, you can pass in an application name prefix such that all environment variables
// will need to start with PREFIX_ to be picked up as valid environment variables. For example,
// if you specified the prefix as "availability", then the program would only detect environment
// variables like "AVAILABILITY_KAFKA_BROKER" and not "KAFKA_BROKER". There is no need to
// capitalize the prefix name.
//
// Note: CLI arguments (eg --address=localhost) will always take precedence over environment variables
func CobraBindEnvironmentVariables(prefix string) func(cmd *cobra.Command, _ []string) {
	// Search for environment values with the given prefix
	viper.SetEnvPrefix(prefix)
	// Automatically extract values from Cobra pflags as prefixed above
	viper.AutomaticEnv()

	return func(cmd *cobra.Command, _ []string) {
		// Provide flags to Viper for environment variable overrides
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if !f.Changed {
				underscoredName := strings.Replace(f.Name, "-", "_", -1)
				if viper.IsSet(underscoredName) {
					strV := viper.GetString(underscoredName)
					cmd.Flags().Set(f.Name, strV)
				}
			}
		})
	}
}
