package vaultunboxer

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type BoxedValue struct {
	vaultPath string
	dataPath  []string
	useCache  bool
}

const (
	prefixNormal  = "VAULT["
	prefixNocache = "VAULT_NOCACHE["
	suffix        = "]"
)

var (
	ErrNotABoxedValue    = errors.New("doesn't look like a Vault-boxed value")
	ErrInvalidBoxedValue = errors.New("Vault-boxed value has an invalid format")
)

func NewBoxedValue(rawBoxedValue string) (*BoxedValue, error) {
	var useCache bool

	if trimmed := strings.TrimPrefix(rawBoxedValue, prefixNormal); trimmed != rawBoxedValue {
		rawBoxedValue = trimmed
		useCache = true
	} else if trimmed := strings.TrimPrefix(rawBoxedValue, prefixNocache); trimmed != rawBoxedValue {
		rawBoxedValue = trimmed
		useCache = false
	} else {
		return nil, ErrNotABoxedValue
	}

	if trimmed := strings.TrimSuffix(rawBoxedValue, suffix); trimmed != rawBoxedValue {
		rawBoxedValue = trimmed
	} else {
		return nil, ErrNotABoxedValue
	}

	parts := strings.Split(rawBoxedValue, " ")
	if len(parts) != 2 {
		return nil, fmt.Errorf("%w: there should be 2 parameters (path and a selector), found %d",
			ErrInvalidBoxedValue, len(parts))
	}

	dataPath := strings.Split(parts[1], ".")

	for _, element := range dataPath {
		if element == "" {
			return nil, fmt.Errorf("%w: found an empty selector element ", ErrInvalidBoxedValue)
		}
	}

	return &BoxedValue{
		vaultPath: parts[0],
		dataPath:  dataPath,
		useCache:  useCache,
	}, nil
}

func (value *BoxedValue) UseCache() bool {
	return value.useCache
}

func (value *BoxedValue) Select(data interface{}) (string, error) {
	for _, element := range value.dataPath {
		dataAsMap, ok := data.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("%w: selector's element %q should always "+
				"query in a dictionary/map-like structures", ErrInvalidBoxedValue, element)
		}

		newData, ok := dataAsMap[element]
		if !ok {
			return "", fmt.Errorf("%w: selector's element %q not found in a dictionary/map-like structure",
				ErrInvalidBoxedValue, element)
		}

		data = newData
	}

	switch typedData := data.(type) {
	case string:
		return typedData, nil
	case map[string]interface{}:
		jsonBytes, err := json.Marshal(typedData)
		if err != nil {
			return "", fmt.Errorf("%w: selector's element %q points to a value that cannot be "+
				"marshalled as JSON: %v", ErrInvalidBoxedValue, value.dataPath[len(value.dataPath)-1], err)
		}

		return string(jsonBytes), nil
	default:
		return "", fmt.Errorf("%w: selector's element %q should point to a string, got %T instead",
			ErrInvalidBoxedValue, value.dataPath[len(value.dataPath)-1], typedData)
	}
}
