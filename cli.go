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
	// Search for environment values prefixed with "AVAILABILITY"
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

// RegisterFlags registers HTTP flags with pflags
func (c *HTTPServerConfig) RegisterFlags(flags *pflag.FlagSet, defaultPort int, defaultName, version, appPackage, gitSHA string) {
	c.Version = version
	c.AppPackage = appPackage
	c.GitSHA = gitSHA
	c.Logging.RegisterFlags(flags)
	c.Tracer.RegisterFlags(flags, defaultName)
	flags.StringVarP(&c.Address, "address", "a", "localhost", "Address for server")
	flags.IntVarP(&c.Port, "port", "p", defaultPort, "Port for server")
	flags.StringVar(&c.Name, "server-name", defaultName, "Server Name")
}

// RegisterFlags registers Kafka flags with pflags
func (kc *KafkaConfig) RegisterFlags(flags *pflag.FlagSet) {
	flags.StringVarP(&kc.Broker, "kafka-broker", "b", "kafka:29092", "Kafka broker Address")
	flags.StringVar(&kc.ClientID, "kafka-client-id", "availability", "Kafka consumer Client ID")
	flags.StringVar(&kc.TLSCaCrtPath, "kafka-server-ca-crt-path", "", "Kafka Server TLS CA Certificate Path")
	flags.StringVar(&kc.TLSCrtPath, "kafka-client-crt-path", "", "Kafka Client TLS Certificate Path")
	flags.StringVar(&kc.TLSKeyPath, "kafka-client-key-path", "", "Kafka Client TLS Key Path")
	flags.BoolVar(&kc.Verbose, "kafka-verbose", false, "When this flag is set Kafka will log verbosely")
	flags.BoolVar(&kc.JSONEnabled, "enable-json", true, "When this flag is set, messages from Kafka will be consumed as JSON instead of Avro")
	flags.StringVar(&kc.KafkaVersion, "kafka-version", "2.1.0", "Kafka broker version")
	flags.StringVar(&kc.ProducerCompressionCodec, "kafka-producer-compression-codec", "none", "Compression codec to use when producing messages, one of: \"none\", \"zstd\", \"snappy\", \"lz4\", \"zstd\", \"gzip\"")
	flags.IntVar(&kc.ProducerCompressionLevel, "kafka-producer-compression-level", -1000, "Compression level to use on produced messages, -1000 signifies to use the default level.")
	kc.SchemaRegistry = &SchemaRegistryConfig{}
	kc.SchemaRegistry.RegisterFlags(flags)
}

// RegisterFlags register Logging flags with pflags
func (lc *LoggingConfig) RegisterFlags(flags *pflag.FlagSet) {
	flags.BoolVar(&lc.SentryLoggingEnabled, "sentry-logger-enabled", false, "Send error logs to Sentry")
	flags.BoolVar(&lc.UseDevelopmentLogger, "use-development-logger", true, "Whether to use the development logger")
	flags.StringArrayVar(&lc.OutputPaths, "log-output-paths", []string{}, "Log file path for standard logging. Logs always output to stdout.")
	flags.StringArrayVar(&lc.ErrorOutputPaths, "log-error-output-paths", []string{}, "Log file path for error logging. Error logs always output to stderr.")
	flags.StringVar(&lc.Level, "log-level", "info", "Application log level")
	flags.IntVar(&lc.SamplingInitial, "log-sampling-initial", 100, "Number of log messages at given level and message to keep each second. Only valid when not using the development logger.")
	flags.IntVar(&lc.SamplingThereafter, "log-sampling-thereafter", 100, "Keep every Nth log with a given message and threshold after log-sampling-initial is exceeded. Only valid when not using the development logger.")
}

// RegisterFlags registers Kafka flags with pflags
func (src *SchemaRegistryConfig) RegisterFlags(flags *pflag.FlagSet) {
	flags.StringVarP(&src.SchemaRegistryURL, "kafka-schema-registry", "r", "http://localhost:8081", "Kafka Schema Registry Address")
}

// RegisterFlags registers Sentry flags with pflags
func (sc *SentryConfig) RegisterFlags(flags *pflag.FlagSet) {
	flags.StringVar(&sc.DSN, "sentry-dsn", "", "Sentry DSN")
}

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

// RegisterFlags registers AWS config flags with pflags
func (a *AWSConfig) RegisterFlags(flags *pflag.FlagSet, defaultRegion string) {
	flags.StringVar(&a.Region, "aws-region", defaultRegion, "AWS region to connect to, if any")
	flags.StringVar(&a.Profile, "aws-profile", "", "AWS profile to assume, if any")
}
