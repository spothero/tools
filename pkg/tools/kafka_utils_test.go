package core

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test that all supported types are handled correctly
func TestUnmarshalMap(t *testing.T) {
	// Define a struct containing every supported type
	type unmarshalTarget struct {
		A int       `kafka:"a"`
		B int8      `kafka:"b"`
		C int16     `kafka:"c"`
		D int32     `kafka:"d"`
		E int64     `kafka:"e"`
		F uint      `kafka:"f"`
		G uint8     `kafka:"g"`
		H uint16    `kafka:"h"`
		I uint32    `kafka:"i"`
		J uint64    `kafka:"j"`
		K bool      `kafka:"k"`
		L string    `kafka:"l"`
		M time.Time `kafka:"m"`
		N float32   `kafka:"n"`
		O float64   `kafka:"o"`
		P bool      `kafka:"p"`
		Q time.Time `kafka:"q"`
	}
	target := &unmarshalTarget{}

	// Build a dummy decoded Kafka message
	kafkaConnectMessage := make(map[string]interface{})
	kafkaConnectMessage["a"] = int32(1)
	kafkaConnectMessage["b"] = int32(2)
	kafkaConnectMessage["c"] = int32(3)
	kafkaConnectMessage["d"] = int32(4)
	kafkaConnectMessage["e"] = int32(5)
	kafkaConnectMessage["f"] = int32(6)
	kafkaConnectMessage["g"] = int32(7)
	kafkaConnectMessage["h"] = int32(8)
	kafkaConnectMessage["i"] = int32(9)
	kafkaConnectMessage["j"] = int32(10)
	kafkaConnectMessage["k"] = int32(0)
	kafkaConnectMessage["l"] = "abc"
	kafkaConnectMessage["m"] = time.Unix(1522083600, 0).UTC().Unix() * 1000
	kafkaConnectMessage["n"] = float32(1.1)
	kafkaConnectMessage["o"] = float64(2.2)
	kafkaConnectMessage["p"] = true
	kafkaConnectMessage["q"] = "2018-08-23T15:56:00-05:00"

	messageDecoder := kafkaMessageDecoder{}
	errs := messageDecoder.unmarshalKafkaMessageMap(kafkaConnectMessage, target)
	assert.Empty(t, errs)
	assert.Equal(t, 1, target.A)
	assert.Equal(t, int8(2), target.B)
	assert.Equal(t, int16(3), target.C)
	assert.Equal(t, int32(4), target.D)
	assert.Equal(t, int64(5), target.E)
	assert.Equal(t, uint(6), target.F)
	assert.Equal(t, uint8(7), target.G)
	assert.Equal(t, uint16(8), target.H)
	assert.Equal(t, uint32(9), target.I)
	assert.Equal(t, uint64(10), target.J)
	assert.False(t, target.K)
	assert.Equal(t, "abc", target.L)
	assert.Equal(t, time.Unix(1522083600, 0), target.M)
	assert.Equal(t, float32(1.1), target.N)
	assert.Equal(t, float64(2.2), target.O)
	assert.Equal(t, true, target.P)
	central, _ := time.LoadLocation("America/Chicago")
	assert.Equal(t, time.Date(2018, 8, 23, 15, 56, 0, 0, central).UTC(), target.Q.UTC())
}

// Test that nullable fields from Kafka Connect are handled correctly
func TestUnmarshalMap_NullableFields(t *testing.T) {
	type unmarshalTarget struct {
		A int `kafka:"a"`
	}
	target := &unmarshalTarget{}

	message := make(map[string]interface{})
	nullable := make(map[string]interface{})
	nullable["int"] = int32(123)
	message["a"] = nullable

	messageDecoder := kafkaMessageDecoder{}
	errs := messageDecoder.unmarshalKafkaMessageMap(message, target)
	assert.Empty(t, errs)
	assert.Equal(t, 123, target.A)
}

// Test that nil fields work
func TestUnmarshalMap_NilFields(t *testing.T) {
	type unmarshalTarget struct {
		A int `kafka:"a"`
	}
	target := &unmarshalTarget{}
	message := make(map[string]interface{})
	messageDecoder := kafkaMessageDecoder{}
	errs := messageDecoder.unmarshalKafkaMessageMap(message, target)
	assert.Empty(t, errs)
	assert.Equal(t, 0, target.A)
}

// Test that unsupported field types return errors
func TestUnmarshalMap_UnsupportedType(t *testing.T) {
	type unmarshalTarget struct {
		A []byte `kafka:"a"`
	}
	target := &unmarshalTarget{}

	message := make(map[string]interface{})
	message["a"] = []byte{'T', 'H', 'A', 'N', 'K'}

	messageDecoder := kafkaMessageDecoder{}
	errs := messageDecoder.unmarshalKafkaMessageMap(message, target)
	require.Len(t, errs, 1)
	expectedErr := fmt.Errorf("unhandled Avro type []uint8, field with tag a will not be set")
	assert.Equal(t, expectedErr, errs[0])
}

// Test that fields that can't be set return errors
func TestUnmarshalMap_UnexportedField(t *testing.T) {
	type unmarshalTarget struct {
		a int `kafka:"a"`
	}
	target := &unmarshalTarget{}

	message := make(map[string]interface{})
	message["a"] = 1

	messageDecoder := kafkaMessageDecoder{}
	errs := messageDecoder.unmarshalKafkaMessageMap(message, target)
	require.Len(t, errs, 1)
	expectedErr := fmt.Errorf("cannot set invalid field with tag a")
	assert.Equal(t, expectedErr, errs[0])
}
