package model

import (
	"fmt"
	"strconv"

	"github.com/m-mizutani/goerr/v2"
)

type ActionResultType string

const (
	ActionResultTypeString ActionResultType = "string"
	ActionResultTypeJSON   ActionResultType = "json"
	ActionResultTypeCSV    ActionResultType = "csv"
	ActionResultTypeTSV    ActionResultType = "tsv"
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
	Choices     ChoiceSpecs
}

type ChoiceSpecs []ChoiceSpec

func (x ChoiceSpecs) has(value string) bool {
	for _, choice := range x {
		if choice.Value == value {
			return true
		}
	}
	return false
}

type ChoiceSpec struct {
	Value       string
	Description string
}

type Arguments map[string]any

// findArgumentSpec is kept for future use when implementing argument validation
func findArgumentSpec(name string, spec []ArgumentSpec) *ArgumentSpec {
	for _, s := range spec {
		if s.Name == name {
			return &s
		}
	}
	return nil
}

func (x ActionSpec) Validate(args Arguments) error {
	// Check if all required arguments are present
	for _, arg := range x.Args {
		if arg.Required && args[arg.Name] == nil {
			return goerr.New("required argument is missing", goerr.V("name", arg.Name))
		}
	}

	// Check if all choices are valid
	for _, arg := range x.Args {
		if arg.Choices != nil {
			if args[arg.Name] == nil {
				if arg.Required {
					return goerr.New("required argument is missing", goerr.V("name", arg.Name))
				}
				continue
			}

			if !arg.Choices.has(args[arg.Name].(string)) {
				return goerr.New("invalid choice", goerr.V("name", arg.Name), goerr.V("value", args[arg.Name]))
			}
		}
	}

	// Check if all arguments are valid
	for _, arg := range x.Args {
		if args[arg.Name] == nil {
			continue
		}

		switch arg.Type {
		case ArgumentTypeString:
			if args[arg.Name] == "" {
				return goerr.New("invalid argument", goerr.V("name", arg.Name))
			}

		case ArgumentTypeNumber:
			v, ok := args[arg.Name].(float64)
			if !ok {
				return goerr.New("invalid argument type", goerr.V("name", arg.Name))
			}
			if _, err := strconv.ParseFloat(fmt.Sprint(v), 64); err != nil {
				return goerr.New("invalid argument", goerr.V("name", arg.Name))
			}

		case ArgumentTypeBoolean:
			_, ok := args[arg.Name].(bool)
			if !ok {
				return goerr.New("invalid argument type", goerr.V("name", arg.Name))
			}
		}
	}

	return nil
}
