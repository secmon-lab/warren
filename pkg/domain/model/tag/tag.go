package tag

import "time"

// Tag represents a tag name
type Tag string

// Set represents a set of tags using map for O(1) operations
type Set map[Tag]bool

// Metadata represents metadata about a tag
type Metadata struct {
	Name      Tag       `json:"name" firestore:"name"`
	CreatedAt time.Time `json:"created_at" firestore:"createdAt"`
	UpdatedAt time.Time `json:"updated_at" firestore:"updatedAt"`
}

// NewSet creates a new Set from a slice of strings
func NewSet(tags []string) Set {
	ts := make(Set)
	for _, tag := range tags {
		ts[Tag(tag)] = true
	}
	return ts
}

// ToSlice converts Set to a slice of strings
func (ts Set) ToSlice() []string {
	result := make([]string, 0, len(ts))
	for tag := range ts {
		result = append(result, string(tag))
	}
	return result
}

// Add adds a tag to the set
func (ts Set) Add(tag Tag) {
	ts[tag] = true
}

// Remove removes a tag from the set
func (ts Set) Remove(tag Tag) {
	delete(ts, tag)
}

// Has checks if a tag exists in the set
func (ts Set) Has(tag Tag) bool {
	return ts[tag]
}

// Copy creates a copy of the Set
func (ts Set) Copy() Set {
	copied := make(Set)
	for tag := range ts {
		copied[tag] = true
	}
	return copied
}