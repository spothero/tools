package kafka

import (
	"time"

	"github.com/Shopify/sarama"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/pflag"
)

// Config contains all configuration for Kafka message consumption and production
type Config struct {
	sarama.Config
	// Whether or not errors will be returned by the consumer. If true, the consumer errors channel must be read from.
	ConsumerReturnErrors bool
	// Whether or not the producer will return errors. If true, the producer errors channel must be read from.
	ProducerReturnErrors bool
	// Whether or not the producer will return successfully produced messages on the successes channel. If true,
	// the successes channel must be read from.
	ProducerReturnSuccesses bool
	// Prometheus registerer for metrics
	Registerer prometheus.Registerer
	// Frequency with which to collect metrics
	MetricsFrequency time.Duration
	BrokerAddrs      []string
	Verbose          bool
	TLSCaCrtPath     string
	TLSCrtPath       string
	TLSKeyPath       string
	// version to be parsed and loaded into sarama.Config.KafkaVersion
	KafkaVersion string
	// value to be cast to sarama.RequiredAcks and loaded into sarama.Config.Producer.RequiredAcks
	ProducerRequiredAcks int16
	// value to be parsed to sarama.CompressionCodec and loaded into sarama.Config.Producer.CompressionCoded
	ProducerCompressionCodec string
}

// RegisterBaseFlags registers basic Kafka configuration. If using Kafka, these flags should always be registered.
func (c *Config) RegisterBaseFlags(flags *pflag.FlagSet) {
	flags.StringArrayVar(&c.BrokerAddrs, "kafka-broker-addrs", []string{}, "comma-separated list of Kafka broker addresses")
	flags.BoolVar(&c.Verbose, "kafka-verbose", false, "log verbose Kafka client information")
	flags.StringVar(&c.ClientID, "kafka-client-id", "sarama", "client ID to provide to the Kafka broker")
	flags.IntVar(&c.ChannelBufferSize, "kafka-channel-buffer-size", 256, "the number of events to buffer in internal and external channels")
	flags.StringVar(&c.KafkaVersion, "kafka-version", "2.3.0", "the assumed version of Kafka to connect to")
}

// RegisterNetFlags registers configuration for connection to the Kafka broker including TLS configuration.
func (c *Config) RegisterNetFlags(flags *pflag.FlagSet) {
	flags.IntVar(&c.Net.MaxOpenRequests, "kafka-net-max-open-requests", 5, "number of outstanding requests a Kafka connection is allowed to have before it blocks")
	flags.DurationVar(&c.Net.DialTimeout, "kafka-net-dial-timeout", 30*time.Second, "duration to wait for the initial connection to Kafka")
	flags.DurationVar(&c.Net.ReadTimeout, "kafka-net-read-timeout", 30*time.Second, "duration to wait for a response from Kafka")
	flags.DurationVar(&c.Net.WriteTimeout, "kafka-net-write-timeout", 30*time.Second, "duration to wait for transmission to Kafka")
	flags.StringVar(&c.TLSCaCrtPath, "kafka-net-tls-ca-cert", "", "path to the CA certificate for the Kafka broker")
	flags.StringVar(&c.TLSCrtPath, "kafka-net-tls-cert", "", "path to the Kafka TLS client certificate")
	flags.StringVar(&c.TLSKeyPath, "kafka-net-tls-key", "", "path to the Kafka TLS client key")
	flags.DurationVar(&c.Net.KeepAlive, "kafka-net-keep-alive", 0, "the keep-alive period for Kafka network communication")
}

// RegisterMetadataFlags registers configuration for fetching metadata from the Kafka broker.
func (c *Config) RegisterMetadataFlags(flags *pflag.FlagSet) {
	flags.IntVar(&c.Metadata.Retry.Max, "kafka-metadata-retry-max", 3, "the number of times to retry a Kafka metadata request")
	flags.DurationVar(&c.Metadata.Retry.Backoff, "kafka-metadata-retry-backoff", 250*time.Millisecond, "duration to wait when retrying Kafka metadata requests")
	flags.DurationVar(&c.Metadata.RefreshFrequency, "kafka-metadata-refresh-frequency", 10*time.Minute, "frequency with which to refresh Kafka cluster metadata in the background")
	flags.BoolVar(&c.Metadata.Full, "kafka-metadata-full", true, "whether to maintain a full set of Kafka metadata")
	flags.DurationVar(&c.Metadata.Timeout, "kafka-metadata-timeout", 0, "duration to wait for a Kafka metadata response")
}

// RegisterProducerFlags registers configuration options for Kafka producers.
func (c *Config) RegisterProducerFlags(flags *pflag.FlagSet) {
	flags.IntVar(&c.Producer.MaxMessageBytes, "kafka-producer-max-message-bytes", 1000000, "maximum permitted size of produced Kafka messages")
	flags.Int16Var(&c.ProducerRequiredAcks, "kafka-producer-required-acks", 1, "Kafka producer required akcs setting. -1=all, 0=none, 1=local.")
	flags.DurationVar(&c.Producer.Timeout, "kafka-producer-timeout", 10*time.Second, "maximum duration the Kafka broker will wait for the number of required acks. only relevant when the required acks setting is set to all.")
	flags.StringVar(&c.ProducerCompressionCodec, "kafka-producer-compression-codec", "none", "compression coded to use in the Kafka producer, one of \"none\", \"zstd\", \"snappy\", \"lz4\", \"gzip\"")
	flags.IntVar(&c.Producer.CompressionLevel, "kafka-producer-compression-level", -1000, "Kafka producer compression level, -1000 signifies the default level")
	flags.BoolVar(&c.Producer.Idempotent, "kafka-producer-idempotent", false, "enable Kafka producer idempotency")
	flags.IntVar(&c.Producer.Flush.Bytes, "kafka-producer-flush-bytes", 0, "best-effort number of bytes to trigger a flush of the Kafka producer")
	flags.IntVar(&c.Producer.Flush.Messages, "kafka-producer-flush-messages", 0, "best-effort number of messages to trigger a flush of the Kafka producer")
	flags.DurationVar(&c.Producer.Flush.Frequency, "kafka-producer-flush-frequency", 0, "best-effort frequency of flushes of the Kafka producer")
	flags.IntVar(&c.Producer.Flush.MaxMessages, "kafka-producer-flush-max-messages", 0, "maximum number of messages the Kafka producer will send in a single request. 0 signifies unlimited messages.")
	flags.IntVar(&c.Producer.Retry.Max, "kafka-producer-retry-max", 3, "number of times to retry sending a Kafka message")
	flags.DurationVar(&c.Producer.Retry.Backoff, "kafka-producer-retry-backoff", 100*time.Millisecond, "backoff duration between Kafka producer message retries")
}

// RegisterConsumerFlags registers configuration options for Kafka consumers.
func (c *Config) RegisterConsumerFlags(flags *pflag.FlagSet) {
	flags.DurationVar(&c.Consumer.Retry.Backoff, "kafka-consumer-retry-backoff", 2*time.Second, "backoff duration between failed Kafka partition reads")
	flags.Int32Var(&c.Consumer.Fetch.Min, "kafka-consumer-fetch-min", 1, "the minimum number of message bytes to fetch from the Kafka broker")
	flags.Int32Var(&c.Consumer.Fetch.Default, "kafka-consumer-fetch-default", 1000000, "the default number of messages bytes to fetch from the Kafka broker per request")
	flags.Int32Var(&c.Consumer.Fetch.Max, "kafka-consumer-fetch-max", 0, "the maximum number of bytes to fetch from the Kafka broker per request. 0 means no limit.")
	flags.DurationVar(&c.Consumer.MaxWaitTime, "kafka-consumer-max-wait-time", 250*time.Millisecond, "the maximum amount of time the Kafka broker will wait before the min fetch size to be available before returning fewer bytes")
	flags.DurationVar(&c.Consumer.MaxProcessingTime, "kafka-consumer-max-processing-time", 100*time.Millisecond, "the maximum amount of time the Kafka consumer expects a message to process before waiting to fetch more messages")
}

// RegisterConsumerGroupFlags registers options for Kafka consumer group configuration.
func (c *Config) RegisterConsumerGroupFlags(flags *pflag.FlagSet) {
	flags.DurationVar(&c.Consumer.Group.Session.Timeout, "kafka-consumer-group-session-timeout", 10*time.Second, "duration after which if no heartbeats are received by the Kafka broker this consumer will be removed from the group")
	flags.DurationVar(&c.Consumer.Group.Heartbeat.Interval, "kafka-consumer-group-heartbeat-interval", 3*time.Second, "frequency with which to send heartbeats to the Kafka consumer coordinator")
	flags.DurationVar(&c.Consumer.Group.Rebalance.Timeout, "kafka-consumer-group-rebalance-timeout", time.Minute, "maximum allowed time to rejoin the Kafka consumer group after a rebalance was started")
	flags.IntVar(&c.Consumer.Group.Rebalance.Retry.Max, "kafka-consumer-group-rebalance-retry-max", 4, "max number of attempts to try to join a Kafka consumer group")
	flags.DurationVar(&c.Consumer.Group.Rebalance.Retry.Backoff, "kafka-consumer-group-rebalance-retry-backoff", 2*time.Second, "backoff duration between retries to join a Kafka consumer group")
}

// RegisterAdminFlags registers options for the Kafka admin API.
func (c *Config) RegisterAdminFlags(flags *pflag.FlagSet) {
	flags.DurationVar(&c.Admin.Timeout, "kafka-admin-timeout", 3*time.Second, "timeout for the Kafka admin API")
}
