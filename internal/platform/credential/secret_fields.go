package credential

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"
)

func secretFields(raw string) (map[string]string, error) {
	return parseSecretFields(raw, false)
}

func (v *SecretValues) UnmarshalJSON(data []byte) error {
	fields, err := parseSecretFields(string(data), true)
	if err == nil {
		*v = fields
	}
	return err
}

func parseSecretFields(raw string, allowEmpty bool) (map[string]string, error) {
	if len(raw) > 512*1024 {
		return nil, fmt.Errorf("Secret总大小不能超过512KiB")
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return nil, fmt.Errorf("Secret必须是文本键值对象")
	}
	fields := make(map[string]string)
	for decoder.More() {
		token, err = decoder.Token()
		key, ok := token.(string)
		if err != nil || !ok {
			return nil, fmt.Errorf("Secret键名无效")
		}
		if _, exists := fields[key]; exists {
			return nil, fmt.Errorf("Secret键名重复")
		}
		var value interface{}
		if decoder.Decode(&value) != nil {
			return nil, fmt.Errorf("Secret内容无效")
		}
		text, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("Secret内容必须是文本")
		}
		fields[key] = text
	}
	if _, err = decoder.Token(); err != nil {
		return nil, fmt.Errorf("Secret内容无效")
	}
	if decoder.Decode(new(interface{})) != io.EOF || (!allowEmpty && len(fields) == 0) || len(fields) > 30 {
		return nil, fmt.Errorf("Secret需包含1至30个文本键值，总大小不能超过512KiB")
	}
	for key := range fields {
		if key == "" || key == "." || key == ".." || len(key) > 128 || len(validation.IsConfigMapKey(key)) > 0 {
			return nil, fmt.Errorf("Secret键名或内容无效")
		}
	}
	return fields, nil
}

func credentialToken(data string) string {
	sum := sha256.Sum256([]byte(data))
	return hex.EncodeToString(sum[:])
}

func (l *Logic) describeSecret(c *Credential) {
	if c.Type != "secret" || l.box == nil {
		return
	}
	plain, err := l.box.Decrypt(c.Data)
	if err != nil {
		return
	}
	var fields map[string]interface{}
	if json.Unmarshal([]byte(plain), &fields) != nil {
		return
	}
	for key, value := range fields {
		if _, ok := value.(string); ok {
			c.Keys = append(c.Keys, key)
		}
	}
	sort.Strings(c.Keys)
	c.Token = credentialToken(c.Data)
}

func mergeSecretFields(source map[string]string, req *DeriveSecretRequest) (string, error) {
	if strings.TrimSpace(req.Name) == "" || len(req.Name) > 64 || len(req.Retain)+len(req.Values) > 30 {
		return "", fmt.Errorf("参数无效")
	}
	next := make(map[string]string)
	for _, key := range req.Retain {
		value, ok := source[key]
		_, duplicate := next[key]
		if !ok || duplicate {
			return "", fmt.Errorf("保留字段不存在或重复")
		}
		next[key] = value
	}
	for key, value := range req.Values {
		if _, exists := next[key]; exists {
			return "", fmt.Errorf("保留与修改字段重复")
		}
		next[key] = value
	}
	raw, _ := json.Marshal(next)
	if _, err := secretFields(string(raw)); err != nil {
		return "", err
	}
	return string(raw), nil
}
