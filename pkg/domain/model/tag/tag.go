package tag

import (
	"time"
)

// Tag represents a tag name
type Tag string

// Set represents a set of tags using map for O(1) operations
type Set map[Tag]bool

// Metadata represents metadata about a tag
type Metadata struct {
	Name      Tag       `json:"name" firestore:"name"`
	Color     string    `json:"color" firestore:"color"`
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

// FirestoreValue converts Set to map[string]bool for Firestore compatibility
func (ts Set) FirestoreValue() map[string]bool {
	result := make(map[string]bool)
	for tag := range ts {
		result[string(tag)] = true
	}
	return result
}

// FromFirestoreValue creates Set from map[string]bool from Firestore
func FromFirestoreValue(m map[string]bool) Set {
	result := make(Set)
	for tag := range m {
		result[Tag(tag)] = true
	}
	return result
}

// chipColors contains predefined colors suitable for chips/badges
var chipColors = []string{
	"bg-red-100 text-red-800",
	"bg-orange-100 text-orange-800", 
	"bg-amber-100 text-amber-800",
	"bg-yellow-100 text-yellow-800",
	"bg-lime-100 text-lime-800",
	"bg-green-100 text-green-800",
	"bg-emerald-100 text-emerald-800",
	"bg-teal-100 text-teal-800",
	"bg-cyan-100 text-cyan-800",
	"bg-sky-100 text-sky-800",
	"bg-blue-100 text-blue-800",
	"bg-indigo-100 text-indigo-800",
	"bg-violet-100 text-violet-800",
	"bg-purple-100 text-purple-800",
	"bg-fuchsia-100 text-fuchsia-800",
	"bg-pink-100 text-pink-800",
	"bg-rose-100 text-rose-800",
	"bg-slate-100 text-slate-800",
	"bg-gray-100 text-gray-800",
	"bg-zinc-100 text-zinc-800",
}

// GenerateColor generates a deterministic color for a tag name
// Uses FNV-1a hash to ensure same tag names always get the same color
func GenerateColor(tagName string) string {
	h := uint32(2166136261)
	for i := 0; i < len(tagName); i++ {
		h ^= uint32(tagName[i])
		h *= 16777619
	}
	colorIndex := int(h) % len(chipColors)
	return chipColors[colorIndex]
}
