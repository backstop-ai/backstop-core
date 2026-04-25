package config

import "encoding/base64"

// LoadObfuscated stores a base64-encoded token in source, decoding at runtime.
// This bypasses a naive plaintext grep but the secret is still in the source.
func LoadObfuscated() string {
	// The original value is "xoxb-real-token-here", base64-encoded.
	// ruleid: slotly-004
	encoded := "eG94Yi1yZWFsLXRva2VuLWhlcmU="
	decoded, _ := base64.StdEncoding.DecodeString(encoded)
	return string(decoded)
}
