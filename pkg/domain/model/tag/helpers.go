package tag

import (
	"context"
)

// ConvertIDsToNames converts tag IDs to tag names using the provided getter function
func ConvertIDsToNames(ctx context.Context, tagIDs map[string]bool, tagGetter func(context.Context, []string) ([]*Tag, error)) ([]string, error) {
	if len(tagIDs) == 0 {
		return []string{}, nil
	}

	tagIDSlice := make([]string, 0, len(tagIDs))
	for tagID := range tagIDs {
		tagIDSlice = append(tagIDSlice, tagID)
	}

	tags, err := tagGetter(ctx, tagIDSlice)
	if err != nil {
		return nil, err
	}

	tagNames := make([]string, 0, len(tags))
	for _, tag := range tags {
		tagNames = append(tagNames, tag.Name)
	}

	return tagNames, nil
}

// NewTagIDsMap creates a new tag IDs map from a slice
func NewTagIDsMap(tagIDs []string) map[string]bool {
	result := make(map[string]bool)
	for _, tagID := range tagIDs {
		result[tagID] = true
	}
	return result
}

// TagIDsMapToSlice converts tag IDs map to slice
func TagIDsMapToSlice(tagIDs map[string]bool) []string {
	result := make([]string, 0, len(tagIDs))
	for tagID := range tagIDs {
		result = append(result, tagID)
	}
	return result
}
