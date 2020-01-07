package kafka

import (
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_RegisterBaseFlags(t *testing.T) {
	flags := pflag.NewFlagSet("pflags", pflag.PanicOnError)
	c := Config{}
	c.RegisterBaseFlags(flags)
	err := flags.Parse([]string{
		"--kafka-broker-addrs", "1.2.3.4,5.6.7.8",
		"--kafka-client-id", "test",
		"--kafka-channel-buffer-size", "512",
		"--kafka-version", "2.4.0",
		"--verbose",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"1.2.3.4", "5.6.7.8"}, c.BrokerAddrs)
	assert.Equal(t, "test", c.ClientID)
	assert.Equal(t, 512, c.ChannelBufferSize)
	assert.Equal(t, "2.4.0", c.KafkaVersion)
	assert.True(t, c.Verbose)
}

func TestConfig_RegisterNetFlags(t *testing.T) {
	flags := pflag.NewFlagSet("pflags", pflag.PanicOnError)
	c := Config{}
	c.RegisterNetFlags(flags)
	err := flags.Parse([]string{
		"--kafka-net-max-open-requests", "10",
		"--kafka-net-dial-timeout", "1s",
		"--kafka-net-read-timeout", "1s",
		"--kafka-net-write-timeout", "1s",
		"--kafka-net-tls-ca-cert", "ca.cert",
		"--kafka-net-tls-cert", "service.cert",
		"--kafka-net-tls-key", "service.key",
		"--kafka-net-keep-alive", "1s",
	})
	require.NoError(t, err)
	assert.Equal(t, 10, c.Net.MaxOpenRequests)
	assert.Equal(t, time.Second, c.Net.DialTimeout)
	assert.Equal(t, time.Second, c.Net.ReadTimeout)
	assert.Equal(t, time.Second, c.Net.WriteTimeout)
	assert.Equal(t, "ca.cert", c.TLSCaCrtPath)
	assert.Equal(t, "service.cert", c.TLSCrtPath)
	assert.Equal(t, "service.key", c.TLSKeyPath)
	assert.Equal(t, time.Second, c.Net.KeepAlive)
}

func TestConfig_RegisterMetadataFlags(t *testing.T) {
	flags := pflag.NewFlagSet("pflags", pflag.PanicOnError)
	c := Config{}
	c.RegisterMetadataFlags(flags)
	err := flags.Parse([]string{
		"--kafka-metadata-retry-max", "5",
		"--kafka-metadata-retry-backoff", "1s",
		"--kafka-metadata-refresh-frequency", "1s",
		"--kafka-metadata-full=false",
		"--kafka-metadata-timeout", "1s",
	})
	require.NoError(t, err)
	assert.Equal(t, 5, c.Metadata.Retry.Max)
	assert.Equal(t, time.Second, c.Metadata.Retry.Backoff)
	assert.Equal(t, time.Second, c.Metadata.RefreshFrequency)
	assert.False(t, c.Metadata.Full)
	assert.Equal(t, time.Second, c.Metadata.Timeout)
}

func TestConfig_RegisterProducerFlags(t *testing.T) {
	flags := pflag.NewFlagSet("pflags", pflag.PanicOnError)
	c := Config{}
	c.RegisterProducerFlags(flags)
	err := flags.Parse([]string{
		"--kafka-producer-max-message-bytes", "555",
		"--kafka-producer-required-acks", "2",
		"--kafka-producer-timeout", "1s",
		"--kafka-producer-compression-codec", "snappy",
		"--kafka-producer-compression-level", "5",
		"--kafka-producer-idempotent",
		"--kafka-producer-flush-bytes", "5",
		"--kafka-producer-flush-messages", "5",
		"--kafka-producer-flush-frequency", "1s",
		"--kafka-producer-flush-max-messages", "5",
		"--kafka-producer-retry-max", "5",
		"--kafka-producer-retry-backoff", "1s",
	})
	require.NoError(t, err)
	assert.Equal(t, 555, c.Producer.MaxMessageBytes)
	assert.Equal(t, int16(2), c.ProducerRequiredAcks)
	assert.Equal(t, time.Second, c.Producer.Timeout)
	assert.Equal(t, "snappy", c.ProducerCompressionCodec)
	assert.Equal(t, 5, c.Producer.CompressionLevel)
	assert.True(t, c.Producer.Idempotent)
	assert.Equal(t, 5, c.Producer.Flush.Bytes)
	assert.Equal(t, 5, c.Producer.Flush.Messages)
	assert.Equal(t, time.Second, c.Producer.Flush.Frequency)
	assert.Equal(t, 5, c.Producer.Flush.MaxMessages)
	assert.Equal(t, 5, c.Producer.Retry.Max)
	assert.Equal(t, time.Second, c.Producer.Retry.Backoff)
}

func TestConfig_RegisterConsumerFlags(t *testing.T) {
	flags := pflag.NewFlagSet("pflags", pflag.PanicOnError)
	c := Config{}
	c.RegisterConsumerFlags(flags)
	err := flags.Parse([]string{
		"--kafka-consumer-retry-backoff", "1s",
		"--kafka-consumer-fetch-min", "5",
		"--kafka-consumer-fetch-default", "5",
		"--kafka-consumer-fetch-max", "5",
		"--kafka-consumer-max-wait-time", "1s",
		"--kafka-consumer-max-processing-time", "1s",
	})
	require.NoError(t, err)
	assert.Equal(t, time.Second, c.Consumer.Retry.Backoff)
	assert.Equal(t, int32(5), c.Consumer.Fetch.Min)
	assert.Equal(t, int32(5), c.Consumer.Fetch.Default)
	assert.Equal(t, int32(5), c.Consumer.Fetch.Max)
	assert.Equal(t, time.Second, c.Consumer.MaxWaitTime)
	assert.Equal(t, time.Second, c.Consumer.MaxProcessingTime)
}

func TestConfig_RegisterConsumerGroupFlags(t *testing.T) {
	flags := pflag.NewFlagSet("pflags", pflag.PanicOnError)
	c := Config{}
	c.RegisterConsumerGroupFlags(flags)
	err := flags.Parse([]string{
		"--kafka-consumer-group-session-timeout", "1s",
		"--kafka-consumer-group-heartbeat-interval", "1s",
		"--kafka-consumer-group-rebalance-timeout", "1s",
		"--kafka-consumer-group-rebalance-retry-max", "5",
		"--kafka-consumer-group-rebalance-retry-backoff", "1s",
	})
	require.NoError(t, err)
	assert.Equal(t, time.Second, c.Consumer.Group.Session.Timeout)
	assert.Equal(t, time.Second, c.Consumer.Group.Heartbeat.Interval)
	assert.Equal(t, time.Second, c.Consumer.Group.Rebalance.Timeout)
	assert.Equal(t, 5, c.Consumer.Group.Rebalance.Retry.Max)
	assert.Equal(t, time.Second, c.Consumer.Group.Rebalance.Retry.Backoff)
}

func TestConfig_RegisterAdminFlags(t *testing.T) {
	flags := pflag.NewFlagSet("pflags", pflag.PanicOnError)
	c := Config{}
	c.RegisterAdminFlags(flags)
	err := flags.Parse([]string{"--kafka-admin-timeout", "1s"})
	require.NoError(t, err)
	assert.Equal(t, time.Second, c.Admin.Timeout)
}
