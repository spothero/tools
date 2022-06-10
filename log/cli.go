// Copyright 2022 SpotHero
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

package log

import "github.com/spf13/pflag"

// RegisterFlags register Logging flags with pflags
func (c *Config) RegisterFlags(flags *pflag.FlagSet) {
	flags.BoolVar(&c.UseDevelopmentLogger, "use-development-logger", true, "Whether to use the development logger")
	flags.StringArrayVar(&c.OutputPaths, "log-output-paths", []string{}, "Log file path for standard logging. Logs always output to stdout.")
	flags.StringArrayVar(&c.ErrorOutputPaths, "log-error-output-paths", []string{}, "Log file path for error logging. Error logs always output to stderr.")
	flags.StringVar(&c.Level, "log-level", "info", "Application log level")
	flags.IntVar(&c.SamplingInitial, "log-sampling-initial", 100, "Number of log messages at given level and message to keep each second. Only valid when not using the development logger.")
	flags.IntVar(&c.SamplingThereafter, "log-sampling-thereafter", 100, "Keep every Nth log with a given message and threshold after log-sampling-initial is exceeded. Only valid when not using the development logger.")
	flags.StringVar(&c.Encoding, "log-encoding", "json", "Log Encoding. One of `console` or `json`")
}
