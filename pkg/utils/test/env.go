package test

import (
	"fmt"
	"os"
	"testing"
)

type EnvVars struct {
	vars map[string]string
}

func NewEnvVars(t *testing.T, keys ...string) EnvVars {
	e := EnvVars{
		vars: map[string]string{},
	}

	for _, key := range keys {
		value, ok := os.LookupEnv(key)
		if !ok {
			t.Skipf("skipping test because %s is not set", key)
		}
		e.vars[key] = value
	}

	return e
}

func (e EnvVars) Get(key string) string {
	if v, ok := e.vars[key]; ok {
		return v
	}

	panic(fmt.Sprintf("env var %s is not set", key))
}
