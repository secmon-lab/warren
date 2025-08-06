package tag

import (
	"time"

	"github.com/secmon-lab/warren/pkg/domain/types"
)

// Set represents a set of tags using map for O(1) operations
type Set map[string]bool

// Tag represents a tag entity with ID-based management
type Tag struct {
	ID          types.TagID `json:"id" firestore:"id"`
	Name        string      `json:"name" firestore:"name"`
	Description string      `json:"description,omitempty" firestore:"description,omitempty"`
	Color       string      `json:"color" firestore:"color"`
	CreatedAt   time.Time   `json:"created_at" firestore:"createdAt"`
	UpdatedAt   time.Time   `json:"updated_at" firestore:"updatedAt"`
	CreatedBy   string      `json:"created_by,omitempty" firestore:"createdBy,omitempty"`
}

// IDSet represents a set of tag IDs using map for O(1) operations
type IDSet map[types.TagID]bool

// NewIDSet creates a new IDSet from a slice of TagIDs
func NewIDSet(tagIDs []types.TagID) IDSet {
	ts := make(IDSet)
	for _, tagID := range tagIDs {
		ts[tagID] = true
	}
	return ts
}

// ToSlice converts IDSet to a slice of TagIDs
func (ts IDSet) ToSlice() []types.TagID {
	result := make([]types.TagID, 0, len(ts))
	for tagID := range ts {
		result = append(result, tagID)
	}
	return result
}

// Add adds a tag ID to the set
func (ts IDSet) Add(tagID types.TagID) {
	ts[tagID] = true
}

// Remove removes a tag ID from the set
func (ts IDSet) Remove(tagID types.TagID) {
	delete(ts, tagID)
}

// Has checks if a tag ID exists in the set
func (ts IDSet) Has(tagID types.TagID) bool {
	return ts[tagID]
}

// Copy creates a copy of the IDSet
func (ts IDSet) Copy() IDSet {
	copied := make(IDSet)
	for tagID := range ts {
		copied[tagID] = true
	}
	return copied
}

// Metadata represents metadata about a tag (deprecated, use Tag instead)
//
// Deprecated: Use Tag struct instead for new implementations
type Metadata struct {
	Name      string    `json:"name" firestore:"name"`
	Color     string    `json:"color" firestore:"color"`
	CreatedAt time.Time `json:"created_at" firestore:"createdAt"`
	UpdatedAt time.Time `json:"updated_at" firestore:"updatedAt"`
}

// NewSet creates a new Set from a slice of strings
func NewSet(tags []string) Set {
	ts := make(Set)
	for _, tag := range tags {
		ts[tag] = true
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
func (ts Set) Add(tag string) {
	ts[tag] = true
}

// Remove removes a tag from the set
func (ts Set) Remove(tag string) {
	delete(ts, tag)
}

// Has checks if a tag exists in the set
func (ts Set) Has(tag string) bool {
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

// ChipColors contains predefined colors suitable for chips/badges
var ChipColors = []string{
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

// ColorNames provides user-friendly color names corresponding to ChipColors
var ColorNames = []string{
	"red",
	"orange", 
	"amber",
	"yellow",
	"lime",
	"green",
	"emerald",
	"teal",
	"cyan",
	"sky",
	"blue",
	"indigo",
	"violet",
	"purple",
	"fuchsia",
	"pink",
	"rose",
	"slate",
	"gray",
	"zinc",
}

// ColorClassToName converts a Tailwind color class to a user-friendly name
func ColorClassToName(colorClass string) string {
	for i, class := range ChipColors {
		if class == colorClass {
			return ColorNames[i]
		}
	}
	return "gray" // fallback
}

// ColorNameToClass converts a user-friendly color name to a Tailwind color class
func ColorNameToClass(colorName string) string {
	for i, name := range ColorNames {
		if name == colorName {
			return ChipColors[i]
		}
	}
	return ChipColors[17] // fallback to gray
}

// GenerateColor generates a deterministic color for a tag name
// Uses FNV-1a hash to ensure same tag names always get the same color
func GenerateColor(tagName string) string {
	h := uint32(2166136261)
	for i := 0; i < len(tagName); i++ {
		h ^= uint32(tagName[i])
		h *= 16777619
	}
	colorIndex := int(h) % len(ChipColors)
	return ChipColors[colorIndex]
}

// MergeTagIDs merges two slices of tag IDs, removing duplicates
func MergeTagIDs(existingTags, newTags []types.TagID) []types.TagID {
	// Create a map to avoid duplicates
	tagMap := make(map[types.TagID]bool)

	// Add existing tags
	for _, tagID := range existingTags {
		tagMap[tagID] = true
	}

	// Add new tags
	for _, tagID := range newTags {
		tagMap[tagID] = true
	}

	// Convert back to slice
	mergedTags := make([]types.TagID, 0, len(tagMap))
	for tagID := range tagMap {
		mergedTags = append(mergedTags, tagID)
	}

	return mergedTags
}
