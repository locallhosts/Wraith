package deploy

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
)

func jsonReader(v any) io.Reader {
	b, err := json.Marshal(v)
	if err != nil {
		// Deploy() only ever passes well-formed structs here; a marshal
		// failure would be a programming error, not a runtime condition
		// callers should need to handle.
		panic(fmt.Sprintf("deploy: failed to marshal document: %v", err))
	}
	return bytes.NewReader(b)
}

func decodeBase64PublicKey(b64 string) (ed25519.PublicKey, error) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, err
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("expected %d-byte ed25519 public key, got %d bytes", ed25519.PublicKeySize, len(raw))
	}
	return ed25519.PublicKey(raw), nil
}
