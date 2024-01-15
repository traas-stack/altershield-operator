package utils

import "os"

type EnvCache struct {
	cache map[string]string
}

func (ec *EnvCache) Get(key string) string {
	value, ok := ec.cache[key]
	if ok {
		return value
	}
	value = os.Getenv(key)
	if value != "" {
		ec.cache[key] = value
	}
	return value
}
