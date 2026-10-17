package credential

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCredentialResponsesNeverSerializeStoredData(t *testing.T) {
	c := Credential{ID: 1, Name: "test-password", Type: "secret", Data: "sensitive-test-content"}
	for _, value := range []interface{}{c, []Credential{c}} {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), c.Data) || strings.Contains(string(data), `"data"`) {
			t.Fatal("credential response includes stored data")
		}
	}
}
