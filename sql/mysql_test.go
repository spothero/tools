package sql

import (
	"context"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
)

func TestConnectionError(t *testing.T) {
	c := &MySQLConfig{Config: mysql.Config{Net: "tcp", Addr: "1.2.3.4:-1"}}
	_, _, err := c.Connect(context.Background(), WithMustRegister(false))
	assert.Error(t, err)
	assert.Equal(t, "dial tcp: address -1: invalid port", err.Error())
}

func TestMySQLConfig_loadCACert(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		expectErr bool
	}{
		{
			"ca certs are loaded",
			"../testdata/fake-ca.pem",
			false,
		}, {
			"bad ca returns an error",
			"../testdata/bad-ca.pem",
			true,
		}, {
			"bad filepath returns an error",
			"../testdata/doesnt-exist.pem",
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := &MySQLConfig{CACertPath: test.path}
			err := c.loadCACert()
			if test.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
