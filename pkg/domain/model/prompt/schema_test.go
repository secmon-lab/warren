package prompt

import (
	"testing"

	"github.com/m-mizutani/gt"
)

func TestToSchema(t *testing.T) {
	t.Run("basic types", func(t *testing.T) {
		type testStruct struct {
			Name    string  `json:"name"`
			Age     int     `json:"age"`
			Score   float64 `json:"score"`
			IsValid bool    `json:"is_valid"`
		}

		schema := ToSchema(testStruct{})
		gt.Value(t, schema.Type).Equal("object")
		gt.Value(t, schema.Properties["name"].Type).Equal("string")
		gt.Value(t, schema.Properties["age"].Type).Equal("integer")
		gt.Value(t, schema.Properties["score"].Type).Equal("number")
		gt.Value(t, schema.Properties["is_valid"].Type).Equal("boolean")
	})

	t.Run("repeated basic types", func(t *testing.T) {
		type testStruct struct {
			Name1 string `json:"name1"`
			Name2 string `json:"name2"`
			Age1  int    `json:"age1"`
			Age2  int    `json:"age2"`
		}

		schema := ToSchema(testStruct{})
		gt.Value(t, schema.Properties["name1"].Type).Equal("string")
		gt.Value(t, schema.Properties["name2"].Type).Equal("string")
		gt.Value(t, schema.Properties["age1"].Type).Equal("integer")
		gt.Value(t, schema.Properties["age2"].Type).Equal("integer")
	})

	t.Run("circular reference", func(t *testing.T) {
		// Define a struct with a circular reference
		type Circular struct {
			Self *Circular `json:"self"`
		}

		schema := ToSchema(Circular{})
		gt.Value(t, schema.Type).Equal("object")
		gt.Value(t, schema.Properties["self"].Type).Equal("object")
		// Check that the circular reference is handled correctly
		gt.Value(t, schema.Properties["self"].Properties["self"].Type).Equal("object")
	})

	t.Run("nested circular reference", func(t *testing.T) {
		// Define a struct with a nested circular reference
		type Node struct {
			Value    string  `json:"value"`
			Children []*Node `json:"children"`
		}

		schema := ToSchema(Node{})
		gt.Value(t, schema.Type).Equal("object")
		gt.Value(t, schema.Properties["value"].Type).Equal("string")
		gt.Value(t, schema.Properties["children"].Type).Equal("array")
		gt.Value(t, schema.Properties["children"].Items.Type).Equal("object")
		// Check that the child node type is handled correctly
		gt.Value(t, schema.Properties["children"].Items.Properties["value"].Type).Equal("string")
		gt.Value(t, schema.Properties["children"].Items.Properties["children"].Type).Equal("array")
	})

	t.Run("complex circular reference", func(t *testing.T) {
		// Define a struct with a more complex circular reference
		type Person struct {
			Name       string    `json:"name"`
			Friends    []*Person `json:"friends"`
			BestFriend *Person   `json:"best_friend"`
		}

		schema := ToSchema(Person{})
		gt.Value(t, schema.Type).Equal("object")
		gt.Value(t, schema.Properties["name"].Type).Equal("string")
		gt.Value(t, schema.Properties["friends"].Type).Equal("array")
		gt.Value(t, schema.Properties["friends"].Items.Type).Equal("object")
		gt.Value(t, schema.Properties["best_friend"].Type).Equal("object")
		// Check that the circular reference in BestFriend is handled correctly
		gt.Value(t, schema.Properties["best_friend"].Properties["name"].Type).Equal("string")
		gt.Value(t, schema.Properties["best_friend"].Properties["friends"].Type).Equal("array")
	})
}
