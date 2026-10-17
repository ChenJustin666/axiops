package credential

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/ChenJustin666/axiops/internal/shared/secretbox"
)

func TestSecretFieldsSupportArbitraryTextAndRejectAmbiguousInput(t *testing.T) {
	valid := `{"API_TOKEN":"test-sk","tls.key":"line1\nline2\n","username":"","access-key":"test-ak"}`
	fields, err := secretFields(valid)
	if err != nil || fields["tls.key"] != "line1\nline2\n" || len(fields) != 4 {
		t.Fatal("arbitrary text fields rejected")
	}
	for _, raw := range []string{`null`, `[]`, `{}`, `{"token":null}`, `{"token":123}`, `{"token":"a","token":"b"}`, `{"bad/key":"a"}`, `{"..":"a"}`, `{"token":"a"} {}`, `{"token":"a"`, strings.Repeat("a", 512*1024+1)} {
		if _, err := secretFields(raw); err == nil {
			t.Error("invalid fields accepted")
		}
	}
}

func TestSecretVersionJSONRejectsAmbiguousReplacementValues(t *testing.T) {
	for _, raw := range []string{`{"values":null}`, `{"values":{"token":null}}`, `{"values":{"token":123}}`, `{"values":{"token":"a","token":"b"}}`} {
		var req DeriveSecretRequest
		if json.Unmarshal([]byte(raw), &req) == nil {
			t.Fatal("ambiguous replacement values accepted")
		}
	}
	for _, raw := range []string{`{"retain":["token"],"values":{}}`, `{"values":{"token":""}}`} {
		var req DeriveSecretRequest
		if err := json.Unmarshal([]byte(raw), &req); err != nil {
			t.Fatal("valid empty replacement rejected", err)
		}
	}
}

func TestSecretVersionRetainsHiddenValuesUpdatesAndDeletesSelectedKeys(t *testing.T) {
	source := map[string]string{"SK": "old-sk", "token": "old-token", "username": "", "remove": "old-remove"}
	raw, err := mergeSecretFields(source, &DeriveSecretRequest{Name: "next", Retain: []string{"SK", "username"}, Values: map[string]string{"token": "new-token", "tls.key": "private\nkey\n"}})
	if err != nil {
		t.Fatal(err)
	}
	next, err := secretFields(raw)
	if err != nil || !reflect.DeepEqual(next, map[string]string{"SK": "old-sk", "token": "new-token", "username": "", "tls.key": "private\nkey\n"}) {
		t.Fatal("incorrect field merge")
	}
	if source["token"] != "old-token" || source["remove"] != "old-remove" {
		t.Fatal("source version modified")
	}
	for _, req := range []DeriveSecretRequest{
		{Name: "x", Retain: []string{"missing"}},
		{Name: "x", Retain: []string{"username", "username"}},
		{Name: "x", Retain: []string{"SK"}, Values: map[string]string{"SK": "x"}},
		{Name: "x"},
	} {
		if _, err := mergeSecretFields(source, &req); err == nil {
			t.Error("invalid retain or empty version accepted")
		}
	}
}

func TestSecretMetadataNeverExposesValues(t *testing.T) {
	box, _ := secretbox.New("test-only-encryption-key")
	data, _ := box.Encrypt(`{"SK":"test-sensitive-sk","tls.key":"test-private-key"}`)
	c := Credential{ID: 1, Type: "secret", Data: data}
	(&Logic{box: box}).describeSecret(&c)
	if !reflect.DeepEqual(c.Keys, []string{"SK", "tls.key"}) || len(c.Token) != 64 {
		t.Fatal("keys or concurrency token missing")
	}
	encoded, _ := json.Marshal(c)
	for _, sensitive := range []string{"test-sensitive-sk", "test-private-key", data} {
		if strings.Contains(string(encoded), sensitive) {
			t.Fatal("secret metadata contains content")
		}
	}
}
