package jose

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/square/go-jose.v2"
)

func TestNewJOSE(t *testing.T) {
	tests := []struct {
		name         string
		keyData      []byte
		statusCode   int
		provideURL   bool
		urlOverride  string
		expectedJWKS *jose.JSONWebKeySet
	}{
		{
			"empty keys array is parsed correctly",
			[]byte("{\"keys\": []}"),
			http.StatusOK,
			true,
			"",
			&jose.JSONWebKeySet{
				Keys: []jose.JSONWebKey{},
			},
		}, {
			"non-200 responses return an error",
			[]byte("{\"keys\": []}"),
			http.StatusNotFound,
			true,
			"",
			nil,
		}, {
			"bad JSON data causes an error",
			[]byte("not json data"),
			http.StatusOK,
			true,
			"",
			nil,
		}, {
			"bad jwks url results in an error",
			[]byte("{\"keys\": []}"),
			http.StatusOK,
			true,
			"badurl",
			nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.statusCode)
				_, _ = w.Write(test.keyData)
				assert.Equal(t, "GET", r.Method)
			}))
			defer ts.Close()
			urls := []string{ts.URL}
			if !test.provideURL {
				urls = []string{}
			}
			if len(test.urlOverride) > 0 {
				urls = []string{test.urlOverride}
			}
			c := &Config{JSONWebKeySetURLs: urls}
			newJOSE := c.NewJOSE()
			assert.Equal(t, test.expectedJWKS, newJOSE.jwks[ts.URL])
		})
	}
}

func TestGenerateClaims(t *testing.T) {
	j := JOSE{
		claimGenerators: []ClaimGenerator{MockGenerator{}},
	}
	assert.Equal(t, []Claim{&MockClaim{}}, j.GetClaims())
}

func TestParseValidateJWT(t *testing.T) {
	// The following is taken from the tests in Square's go-jose
	// See: https://github.com/square/go-jose/tree/8bad6148bd0a8da57f9840693f6f611ac4483d09/jwt
	privateRSAKey := `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDIHBvDHAr7jh8h
xaqBCl11fjI9YZtdC5b3HtXTXZW3c2dIOImNUjffT8POP6p5OpzivmC1om7iOyuZ
3nJjC9LT3zqqs3f2i5d4mImxEuqG6uWdryFfkp0uIv5VkjVO+iQWd6pDAPGP7r1Z
foXCleyCtmyNH4JSkJneNPOk/4BxO8vcvRnCMT/Gv81IT6H+OQ6OovWOuJr8RX9t
1wuCjC9ezZxeI9ONffhiO5FMrVh5H9LJTl3dPOVa4aEcOvgd45hBmvxAyXqf8daE
6Kl2O7vQ4uwgnSTVXYIIjCjbepuersApIMGx/XPSgiU1K3Xtah/TBvep+S3VlwPc
q/QH25S9AgMBAAECggEAe+y8XKYfPw4SxY1uPB+5JSwT3ON3nbWxtjSIYy9Pqp5z
Vcx9kuFZ7JevQSk4X38m7VzM8282kC/ono+d8yy9Uayq3k/qeOqV0X9Vti1qxEbw
ECkG1/MqGApfy4qSLOjINInDDV+mOWa2KJgsKgdCwuhKbVMYGB2ozG2qfYIlfvlY
vLcBEpGWmswJHNmkcjTtGFIyJgPbsI6ndkkOeQbqQKAaadXtG1xUzH+vIvqaUl/l
AkNf+p4qhPkHsoAWXf1qu9cYa2T8T+mEo79AwlgVC6awXQWNRTiyClDJC7cu6NBy
ZHXCLFMbalzWF9qeI2OPaFX2x3IBWrbyDxcJ4TSdQQKBgQD/Fp/uQonMBh1h4Vi4
HlxZdqSOArTitXValdLFGVJ23MngTGV/St4WH6eRp4ICfPyldsfcv6MZpNwNm1Rn
lB5Gtpqpby1dsrOSfvVbY7U3vpLnd8+hJ/lT5zCYt5Eor46N6iWRkYWzNe4PixiF
z1puGUvFCbZdeeACVrPLmW3JKQKBgQDI0y9WTf8ezKPbtap4UEE6yBf49ftohVGz
p4iD6Ng1uqePwKahwoVXKOc179CjGGtW/UUBORAoKRmxdHajHq6LJgsBxpaARz21
COPy99BUyp9ER5P8vYn63lC7Cpd/K7uyMjaz1DAzYBZIeVZHIw8O9wuGNJKjRFy9
SZyD3V0ddQKBgFMdohrWH2QVEfnUnT3Q1rJn0BJdm2bLTWOosbZ7G72TD0xAWEnz
sQ1wXv88n0YER6X6YADziEdQykq8s/HT91F/KkHO8e83zP8M0xFmGaQCOoelKEgQ
aFMIX3NDTM7+9OoUwwz9Z50PE3SJFAJ1n7eEEoYvNfabQXxBl+/dHEKRAoGAPEvU
EaiXacrtg8EWrssB2sFLGU/ZrTciIbuybFCT4gXp22pvXXAHEvVP/kzDqsRhLhwb
BNP6OuSkNziNikpjA5pngZ/7fgZly54gusmW/m5bxWdsUl0iOXVYbeAvPlqGH2me
LP4Pfs1hw17S/cbT9Z1NE31jbavP4HFikeD73SUCgYEArQfuudml6ei7XZ1Emjq8
jZiD+fX6e6BD/ISatVnuyZmGj9wPFsEhY2BpLiAMQHMDIvH9nlKzsFvjkTPB86qG
jCh3D67Os8eSBk5uRC6iW3Fc4DXvB5EFS0W9/15Sl+V5vXAcrNMpYS82OTSMG2Gt
b9Ym/nxaqyTu0PxajXkKm5Q=
-----END PRIVATE KEY-----`
	block, _ := pem.Decode([]byte(privateRSAKey))
	key, _ := x509.ParsePKCS8PrivateKey(block.Bytes)
	rsaKey, _ := key.(*rsa.PrivateKey)
	jwksURL := "test-jwks-url"
	jwks := map[string]*jose.JSONWebKeySet{
		jwksURL: {
			Keys: []jose.JSONWebKey{
				{
					KeyID: "foobar",
					Key:   &rsaKey.PublicKey,
				},
			},
		},
	}

	tests := []struct {
		name         string
		jwt          string
		issuer       string
		jwks         map[string]*jose.JSONWebKeySet
		returnedKeys *jose.JSONWebKeySet
		expectError  bool
	}{
		{
			"valid jwks and token does not produce an error",
			`eyJhbGciOiJSUzI1NiIsImtpZCI6ImZvb2JhciJ9.eyJpc3MiOiJpc3N1ZXIiLCJzY29wZXMiOlsiczEiLCJzMiJdLCJzdWIiOiJzdWJqZWN0In0.RxZhTRfPDb6UJ58FwvC89GgJGC8lAO04tz5iLlBpIJsyPZB0X_UgXSj0SGVFm2jbP_i-ZVH4HFC2fMB1n-so9CnCOpunWwhYNdgF6ewQJ0ADTWwfDGsK12UOmyT2naaZN8ZUBF8cgPtOgdWqQjk2Ng9QFRJxlUuKYczBp7vjWvgX8WMwQcaA-eK7HtguR4e9c4FMbeFK8Soc4jCsVTjIKdSn9SErc42gFu65NI1hZ3OPe_T7AZqdDjCkJpoiJ65GdD_qvGkVndJSEcMp3riXQpAy0JbctVkYecdFaGidbxHRrdcQYHtKn-XGMCh2uoBKleUr1fTMiyCGPQQesy3xHw`,
			"issuer",
			jwks,
			nil,
			false,
		}, {
			"valid token that is not correctly signed produces an error",
			`eyJhbGciOiJSUzI1NiIsImtpZCI6ImZvb2JhcmFiYyJ9.eyJpc3MiOiJpc3N1ZXIiLCJzY29wZXMiOlsiczEiLCJzMiJdLCJzdWIiOiJzdWJqZWN0In0.RxZhTRfPDb6UJ58FwvC89GgJGC8lAO04tz5iLlBpIJsyPZB0X_UgXSj0SGVFm2jbP_i-ZVH4HFC2fMB1n-so9CnCOpunWwhYNdgF6ewQJ0ADTWwfDGsK12UOmyT2naaZN8ZUBF8cgPtOgdWqQjk2Ng9QFRJxlUuKYczBp7vjWvgX8WMwQcaA-eK7HtguR4e9c4FMbeFK8Soc4jCsVTjIKdSn9SErc42gFu65NI1hZ3OPe_T7AZqdDjCkJpoiJ65GdD_qvGkVndJSEcMp3riXQpAy0JbctVkYecdFaGidbxHRrdcQYHtKn-XGMCh2uoBKleUr1fTMiyCGPQQesy3xHw`,
			"issuer",
			jwks,
			nil,
			true,
		}, {
			"invalid jwt produces an error",
			"invalid jwt",
			"issuer",
			jwks,
			nil,
			true,
		}, {
			"lazy load missing keys",
			`eyJhbGciOiJSUzI1NiIsImtpZCI6ImZvb2JhciJ9.eyJpc3MiOiJpc3N1ZXIiLCJzY29wZXMiOlsiczEiLCJzMiJdLCJzdWIiOiJzdWJqZWN0In0.RxZhTRfPDb6UJ58FwvC89GgJGC8lAO04tz5iLlBpIJsyPZB0X_UgXSj0SGVFm2jbP_i-ZVH4HFC2fMB1n-so9CnCOpunWwhYNdgF6ewQJ0ADTWwfDGsK12UOmyT2naaZN8ZUBF8cgPtOgdWqQjk2Ng9QFRJxlUuKYczBp7vjWvgX8WMwQcaA-eK7HtguR4e9c4FMbeFK8Soc4jCsVTjIKdSn9SErc42gFu65NI1hZ3OPe_T7AZqdDjCkJpoiJ65GdD_qvGkVndJSEcMp3riXQpAy0JbctVkYecdFaGidbxHRrdcQYHtKn-XGMCh2uoBKleUr1fTMiyCGPQQesy3xHw`,
			"issuer",
			map[string]*jose.JSONWebKeySet{
				jwksURL: nil,
			},
			&jose.JSONWebKeySet{
				Keys: []jose.JSONWebKey{
					{
						KeyID: "foobar",
						Key:   &rsaKey.PublicKey,
					},
				},
			},
			false,
		}, {
			"lazy load missing keys unsuccessfully",
			`eyJhbGciOiJSUzI1NiIsImtpZCI6ImZvb2JhciJ9.eyJpc3MiOiJpc3N1ZXIiLCJzY29wZXMiOlsiczEiLCJzMiJdLCJzdWIiOiJzdWJqZWN0In0.RxZhTRfPDb6UJ58FwvC89GgJGC8lAO04tz5iLlBpIJsyPZB0X_UgXSj0SGVFm2jbP_i-ZVH4HFC2fMB1n-so9CnCOpunWwhYNdgF6ewQJ0ADTWwfDGsK12UOmyT2naaZN8ZUBF8cgPtOgdWqQjk2Ng9QFRJxlUuKYczBp7vjWvgX8WMwQcaA-eK7HtguR4e9c4FMbeFK8Soc4jCsVTjIKdSn9SErc42gFu65NI1hZ3OPe_T7AZqdDjCkJpoiJ65GdD_qvGkVndJSEcMp3riXQpAy0JbctVkYecdFaGidbxHRrdcQYHtKn-XGMCh2uoBKleUr1fTMiyCGPQQesy3xHw`,
			"issuer",
			map[string]*jose.JSONWebKeySet{
				jwksURL: nil,
			},
			nil,
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				if test.returnedKeys != nil {
					jsonResp, _ := json.Marshal(test.returnedKeys)
					_, _ = w.Write(jsonResp)
				}
				assert.Equal(t, "GET", r.Method)
			}))
			defer ts.Close()

			// Overwrite JWKS URL with test server url
			test.jwks[ts.URL] = test.jwks[jwksURL]
			delete(test.jwks, jwksURL)

			testJOSE := JOSE{jwks: test.jwks, validIssuers: []string{test.issuer}}
			err := testJOSE.ParseValidateJWT(test.jwt)
			if test.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
