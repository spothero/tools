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

package tracing

import "github.com/spf13/pflag"

// RegisterFlags registers Tracer flags with pflags
func (c *Config) RegisterFlags(flags *pflag.FlagSet) {
	flags.BoolVarP(&c.Enabled, "tracer-enabled", "t", true, "Enable tracing")
	flags.StringVar(&c.SamplerType, "tracer-sampler-type", "", "Tracer sampler type")
	flags.Float64Var(&c.SamplerParam, "tracer-sampler-param", 1.0, "Tracer sampler param")
	flags.BoolVar(&c.ReporterLogSpans, "tracer-reporter-log-spans", false, "Tracer Reporter Logs Spans")
	flags.IntVar(&c.ReporterMaxQueueSize, "tracer-reporter-max-queue-size", 100, "Tracer Reporter Max Queue Size")
	flags.DurationVar(&c.ReporterFlushInterval, "tracer-reporter-flush-interval", 1000000000, "Tracer Reporter Flush Interval in nanoseconds")
	flags.StringVar(&c.AgentHost, "tracer-agent-host", "localhost", "Tracer Agent Host")
	flags.IntVar(&c.AgentPort, "tracer-agent-port", 5775, "Tracer Agent Port")
	flags.StringVar(&c.ServiceName, "tracer-service-name", c.ServiceName, "Determines the service name for the Tracer UI")
}
