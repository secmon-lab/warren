package model

import (
	"strconv"

	"github.com/m-mizutani/goerr/v2"
)

type ActionResultType string

const (
	ActionResultTypeString ActionResultType = "string"
	ActionResultTypeJSON   ActionResultType = "json"
	ActionResultTypeCVS    ActionResultType = "cvs"
)

type ActionResult struct {
	Message string
	Type    ActionResultType
	Data    string
}

type ActionSpec struct {
	Name        string
	Description string
	Args        []ArgumentSpec
}

type ArgumentType string

const (
	ArgumentTypeString  ArgumentType = "string"
	ArgumentTypeNumber  ArgumentType = "number"
	ArgumentTypeBoolean ArgumentType = "boolean"
)

type ArgumentSpec struct {
	Name        string
	Description string
	Type        ArgumentType
	Required    bool
	Values      []ValueSpec
}

type ValueSpec struct {
	Value       string
	Description string
}

type Arguments map[string]string

func findArgumentSpec(name string, spec []ArgumentSpec) *ArgumentSpec {
	for _, s := range spec {
		if s.Name == name {
			return &s
		}
	}
	return nil
}

func (x Arguments) Validate(spec []ArgumentSpec) error {
	for k, v := range x {
		arg := findArgumentSpec(k, spec)
		if arg == nil {
			return goerr.New("unknown argument", goerr.V("name", k))
		}

		switch arg.Type {
		case ArgumentTypeBoolean:
			if v != "true" && v != "false" {
				return goerr.New("invalid boolean argument", goerr.V("name", k), goerr.V("value", v))
			}
		case ArgumentTypeNumber:
			if _, err := strconv.ParseFloat(v, 64); err != nil {
				return goerr.New("invalid number argument", goerr.V("name", k), goerr.V("value", v))
			}
		case ArgumentTypeString:
			if v == "" {
				return goerr.New("invalid string argument", goerr.V("name", k), goerr.V("value", v))
			}
		}

		if !arg.Required && v == "" {
			continue
		}

		if v == "" {
			return goerr.New("missing argument", goerr.V("name", k))
		}
	}

	for _, s := range spec {
		if _, ok := x[s.Name]; !ok && s.Required {
			return goerr.New("missing argument", goerr.V("name", s.Name))
		}
	}

	return nil
}
