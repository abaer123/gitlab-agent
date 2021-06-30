package filez

import (
	"encoding/base64"
	"fmt"
	"os"
)

func LoadBase64Secret(filename string) ([]byte, error) {
	encodedAuthSecret, err := os.ReadFile(filename) // nolint: gosec
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	decodedAuthSecret := make([]byte, len(encodedAuthSecret))

	n, err := base64.StdEncoding.Decode(decodedAuthSecret, encodedAuthSecret)
	if err != nil {
		return nil, fmt.Errorf("decoding: %w", err)
	}
	return decodedAuthSecret[:n], nil
}
