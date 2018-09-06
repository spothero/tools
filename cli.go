package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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

// RegisterViperFlags registers HTTP flags with Viper CLIs
func (c *HTTPServerConfig) RegisterViperFlags(flags *pflag.FlagSet, defaultPort int) {
	flags.StringVarP(&c.Address, "address", "a", "localhost", "Address for server")
	flags.IntVarP(&c.Port, "port", "p", defaultPort, "Port for server")
	flags.StringVar(&c.Name, "server-name", c.Name, "Server Name")
}

// RegisterViperFlags registers Kafka flags with Viper CLIs
func (kc *KafkaConfig) RegisterViperFlags(flags *pflag.FlagSet) {
	flags.StringVarP(&kc.Broker, "kafka-broker", "b", "kafka:29092", "Kafka Broker Address")
	flags.StringVar(&kc.ClientID, "kafka-client-id", "availability", "Kafka consumer Client ID")
	flags.StringVar(&kc.TLSCaCrtPath, "kafka-server-ca-crt-path", "", "Kafka Server TLS CA Certificate Path")
	flags.StringVar(&kc.TLSCrtPath, "kafka-client-crt-path", "", "Kafka Client TLS Certificate Path")
	flags.StringVar(&kc.TLSKeyPath, "kafka-client-key-path", "", "Kafka Client TLS Key Path")
	flags.BoolVar(&kc.Verbose, "kafka-verbose", false, "When this flag is set Kafka will log verbosely")
	flags.BoolVar(&kc.JSONEnabled, "enable-json", true, "When this flag is set, messages from Kafka will be consumed as JSON instead of Avro")
}

// RegisterViperFlags register Logging flags with Viper CLIs
func (lc *LoggingConfig) RegisterViperFlags(flags *pflag.FlagSet) {
	flags.BoolVar(&lc.SentryLoggingEnabled, "sentry-logger-enabled", false, "Send error logs to Sentry")
	flags.BoolVar(&lc.UseDevelopmentLogger, "use-development-logger", true, "Whether to use the development logger")
	flags.StringArrayVar(&lc.OutputPaths, "log-output-paths", []string{}, "Log file path for standard logging. Logs always output to stdout.")
	flags.StringArrayVar(&lc.ErrorOutputPaths, "log-error-output-paths", []string{}, "Log file path for error logging. Error logs always output to stderr.")
	flags.StringVar(&lc.Level, "log-level", "info", "Application log level")
	flags.IntVar(&lc.SamplingInitial, "log-sampling-initial", 100, "Number of log messages at given level and message to keep each second. Only valid when not using the development logger.")
	flags.IntVar(&lc.SamplingThereafter, "log-sampling-thereafter", 100, "Keep every Nth log with a given message and threshold after log-sampling-initial is exceeded. Only valid when not using the development logger.")
}

// RegisterViperFlags registers Sentry flags with Viper CLIs
func (nc *NewRelicConfig) RegisterViperFlags(flags *pflag.FlagSet, appName, environment string) {
	flags.BoolVar(&nc.Enabled, "new-relic-enabled", false, "Enable New Relic metrics")
	flags.StringVar(&nc.LicenseKey, "new-relic-license-key", "", "New Relic license key")
	flags.StringVar(&nc.AppName, "new-relic-app-name", fmt.Sprintf("%s (%s)", appName, environment), "New Relic app name")
}

// RegisterViperFlags registers Kafka flags with Viper CLIs
func (src *SchemaRegistryConfig) RegisterViperFlags(flags *pflag.FlagSet) {
	flags.StringVarP(&src.SchemaRegistryURL, "kafka-schema-registry", "r", "http://localhost:8081", "Kafka Schema Registry Address")
}

// RegisterViperFlags registers Sentry flags with Viper CLIs
func (sc *SentryConfig) RegisterViperFlags(flags *pflag.FlagSet) {
	flags.StringVar(&sc.DSN, "sentry-dsn", "", "Sentry DSN")
}

// RegisterViperFlags registers Tracer flags with Viper CLIs
func (tc *TracingConfig) RegisterViperFlags(flags *pflag.FlagSet, defaultTracerName string) {
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

// RegisterViperFlags registers Kubernetes flags with Viper CLIs
func (kc *KubernetesConfig) RegisterViperFlags(flags *pflag.FlagSet) {
	homePath := ""
	home := os.Getenv("HOME")
	if home != "" {
		homePath = filepath.Join(home, ".kube", "config")
	}

	flags.StringVar(&kc.ConfigPath, "kube-config-path", homePath, "Absolute path to the kubeconfig file. Leave blank in cluster.")
}

// RegisterViperFlags registers RemoteCache flags with Viper CLIs
func (rcc *RemoteCacheConfig) RegisterViperFlags(flags *pflag.FlagSet) {
	defaultURLs := []string{
		"127.0.0.1:7000",
		"127.0.0.1:7001",
		"127.0.0.1:7002",
		"127.0.0.1:7003",
		"127.0.0.1:7004",
		"127.0.0.1:7005",
	}
	flags.StringSliceVar(&rcc.URLs, "cache-urls", defaultURLs, "Remote Clustered Redis Cache URLs as a comma-separated list")
	flags.StringVar(&rcc.AuthToken, "cache-auth-token", "", "Redis Auth Token, If Any")
	flags.DurationVar(&rcc.Timeout, "cache-timeout", time.Duration(time.Second*5), "Remote Redis Cache Connection Timeout")
}

// RegisterViperFlags registers LocalCache flags with Viper CLIs
func (lcc *LocalCacheConfig) RegisterViperFlags(flags *pflag.FlagSet) {
	flags.DurationVar(&lcc.Eviction, "cache-eviction", time.Duration(time.Second*5), "How frequently to evict from cache")
	flags.DurationVar(&lcc.TTL, "cache-ttl", time.Duration(time.Minute*60), "Cache Entry TTL for local cache")
	flags.UintVar(&lcc.Shards, "cache-shards", 0, "Number of shards for local cluster. 0 means the program decides itself. Must be power of 2.")
}

// RegisterViperFlags registers TieredCache flags with Viper CLIs
func (tcc *TieredCacheConfig) RegisterViperFlags(flags *pflag.FlagSet) {
	tcc.RemoteConfig.RegisterViperFlags(flags)
	tcc.LocalConfig.RegisterViperFlags(flags)
}
