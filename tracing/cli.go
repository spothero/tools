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
func (tc *TracingConfig) RegisterFlags(flags *pflag.FlagSet, defaultTracerName string) {
	flags.BoolVarP(&tc.Enabled, "tracer-enabled", "t", true, "Enable tracing")
	flags.StringVar(&tc.SamplerType, "tracer-sampler-type", "", "Tracer sampler type")
	flags.Float64Var(&tc.SamplerParam, "tracer-sampler-param", 1.0, "Tracer sampler param")
	flags.BoolVar(&tc.ReporterLogSpans, "tracer-reporter-log-spans", false, "Tracer Reporter Logs Spans")
	flags.IntVar(&tc.ReporterMaxQueueSize, "tracer-reporter-max-queue-size", 100, "Tracer Reporter Max Queue Size")
	flags.DurationVar(&tc.ReporterFlushInterval, "tracer-reporter-flush-interval", 1000000000, "Tracer Reporter Flush Interval in nanoseconds")
	flags.StringVar(&tc.AgentHost, "tracer-agent-host", "localhost", "Tracer Agent Host")
	flags.IntVar(&tc.AgentPort, "tracer-agent-port", 5775, "Tracer Agent Port")
	flags.StringVar(&tc.ServiceName, "tracer-service-name", defaultTracerName, "Determines the service name for the Tracer UI")
}
