package encode

import (
	"encoding/base64"
)

func DecodeBase64String(value string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(value)
}

func EncodeBase64String(value []byte) string {
	return base64.StdEncoding.EncodeToString(value)
}
