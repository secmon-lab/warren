package embedding

import (
	"math"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/gt"
)

func TestWeightedAverage(t *testing.T) {
	type testCase struct {
		name          string
		embeddings    []firestore.Vector32
		weights       []float32
		expected      firestore.Vector32
		expectedError bool
	}

	runTest := func(tc testCase) func(t *testing.T) {
		return func(t *testing.T) {
			result, err := WeightedAverage(tc.embeddings, tc.weights)

			if tc.expectedError {
				gt.Error(t, err)
				return
			}

			gt.NoError(t, err)

			if len(tc.expected) == 0 {
				gt.Array(t, result).Length(0)
				return
			}

			gt.Array(t, result).Length(len(tc.expected))
			for i, expected := range tc.expected {
				// Use approximate equality for float32 comparison
				diff := math.Abs(float64(result[i] - expected))
				gt.Number(t, diff).Less(0.001)
			}
		}
	}

	t.Run("basic weighted average", runTest(testCase{
		name: "basic weighted average",
		embeddings: []firestore.Vector32{
			{1.0, 2.0, 3.0},
			{4.0, 5.0, 6.0},
		},
		weights:       []float32{0.3, 0.7},
		expected:      firestore.Vector32{3.1, 4.1, 5.1}, // 1.0*0.3 + 4.0*0.7 = 3.1, 2.0*0.3 + 5.0*0.7 = 4.1, 3.0*0.3 + 6.0*0.7 = 5.1
		expectedError: false,
	}))

	t.Run("equal weights", runTest(testCase{
		name: "equal weights",
		embeddings: []firestore.Vector32{
			{1.0, 2.0, 3.0},
			{3.0, 4.0, 5.0},
		},
		weights:       []float32{0.5, 0.5},
		expected:      firestore.Vector32{2.0, 3.0, 4.0},
		expectedError: false,
	}))

	t.Run("single embedding", runTest(testCase{
		name: "single embedding",
		embeddings: []firestore.Vector32{
			{1.0, 2.0, 3.0},
		},
		weights:       []float32{1.0},
		expected:      firestore.Vector32{1.0, 2.0, 3.0},
		expectedError: false,
	}))

	t.Run("empty embeddings", runTest(testCase{
		name:          "empty embeddings",
		embeddings:    []firestore.Vector32{},
		weights:       []float32{},
		expected:      firestore.Vector32{},
		expectedError: true, // Now expects error
	}))

	t.Run("mismatched lengths", runTest(testCase{
		name: "mismatched lengths",
		embeddings: []firestore.Vector32{
			{1.0, 2.0, 3.0},
		},
		weights:       []float32{0.5, 0.5}, // Different length
		expected:      firestore.Vector32{},
		expectedError: true, // Now expects error
	}))

	t.Run("zero dimension embeddings", runTest(testCase{
		name:          "zero dimension embeddings",
		embeddings:    []firestore.Vector32{{}},
		weights:       []float32{1.0},
		expected:      firestore.Vector32{},
		expectedError: true, // Now expects error
	}))

	t.Run("different dimension embeddings", runTest(testCase{
		name: "different dimension embeddings",
		embeddings: []firestore.Vector32{
			{1.0, 2.0, 3.0},
			{4.0, 5.0}, // Different dimension
		},
		weights:       []float32{0.5, 0.5},
		expected:      firestore.Vector32{},
		expectedError: true, // Now expects error
	}))

	t.Run("ticket-like scenario", runTest(testCase{
		name: "ticket-like scenario (metadata 0.3, alerts 0.7)",
		embeddings: []firestore.Vector32{
			{1.0, 0.0, 0.0}, // metadata embedding
			{0.0, 1.0, 0.0}, // alert embedding
		},
		weights:       []float32{0.3, 0.7},
		expected:      firestore.Vector32{0.3, 0.7, 0.0},
		expectedError: false,
	}))
}

func TestAverage(t *testing.T) {
	type testCase struct {
		name       string
		embeddings []firestore.Vector32
		expected   firestore.Vector32
	}

	runTest := func(tc testCase) func(t *testing.T) {
		return func(t *testing.T) {
			result := Average(tc.embeddings)

			if len(tc.expected) == 0 {
				gt.Array(t, result).Length(0)
				return
			}

			gt.Array(t, result).Length(len(tc.expected))
			for i, expected := range tc.expected {
				// Use approximate equality for float32 comparison
				diff := math.Abs(float64(result[i] - expected))
				gt.Number(t, diff).Less(0.001)
			}
		}
	}

	t.Run("basic average", runTest(testCase{
		name: "basic average",
		embeddings: []firestore.Vector32{
			{1.0, 2.0, 3.0},
			{3.0, 4.0, 5.0},
		},
		expected: firestore.Vector32{2.0, 3.0, 4.0},
	}))

	t.Run("empty embeddings", runTest(testCase{
		name:       "empty embeddings",
		embeddings: []firestore.Vector32{},
		expected:   firestore.Vector32{},
	}))
}
