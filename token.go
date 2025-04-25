package page

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

func EncodePageToken(data any) string {
	jsonByte, _ := json.Marshal(data)
	return base64.StdEncoding.EncodeToString(jsonByte)
}

func DecodePageToken[T any](token string) (*T, error) {
	decoded, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor: %w", err)
	}

	var res T
	if err = json.Unmarshal(decoded, &res); err != nil {
		return nil, err
	}
	return &res, nil
}
