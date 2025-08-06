package tag_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/tag"
)

func TestConvertIDsToNames(t *testing.T) {
	ctx := context.Background()

	t.Run("empty tag IDs", func(t *testing.T) {
		tagIDs := map[string]bool{}
		tagGetter := func(ctx context.Context, ids []string) ([]*tag.Tag, error) {
			return []*tag.Tag{}, nil
		}

		result, err := tag.ConvertIDsToNames(ctx, tagIDs, tagGetter)
		gt.NoError(t, err)
		gt.A(t, result).Equal([]string{})
	})

	t.Run("valid tag IDs", func(t *testing.T) {
		tagIDs := map[string]bool{
			"tag_1": true,
			"tag_2": true,
		}
		tagGetter := func(ctx context.Context, ids []string) ([]*tag.Tag, error) {
			return []*tag.Tag{
				{ID: "tag_1", Name: "Tag One"},
				{ID: "tag_2", Name: "Tag Two"},
			}, nil
		}

		result, err := tag.ConvertIDsToNames(ctx, tagIDs, tagGetter)
		gt.NoError(t, err)
		gt.A(t, result).Length(2)
		// Check that both tag names are present (order doesn't matter)
		containsTagOne := false
		containsTagTwo := false
		for _, name := range result {
			if name == "Tag One" {
				containsTagOne = true
			}
			if name == "Tag Two" {
				containsTagTwo = true
			}
		}
		gt.Value(t, containsTagOne).Equal(true)
		gt.Value(t, containsTagTwo).Equal(true)
	})
}

func TestNewTagIDsMap(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		result := tag.NewTagIDsMap([]string{})
		gt.V(t, len(result)).Equal(0)
	})

	t.Run("valid tag IDs", func(t *testing.T) {
		tagIDs := []string{"tag_1", "tag_2", "tag_3"}
		result := tag.NewTagIDsMap(tagIDs)
		gt.V(t, len(result)).Equal(3)
		gt.Value(t, result["tag_1"]).Equal(true)
		gt.Value(t, result["tag_2"]).Equal(true)
		gt.Value(t, result["tag_3"]).Equal(true)
	})

	t.Run("duplicate tag IDs", func(t *testing.T) {
		tagIDs := []string{"tag_1", "tag_2", "tag_1"}
		result := tag.NewTagIDsMap(tagIDs)
		gt.V(t, len(result)).Equal(2)
		gt.Value(t, result["tag_1"]).Equal(true)
		gt.Value(t, result["tag_2"]).Equal(true)
	})
}

func TestTagIDsMapToSlice(t *testing.T) {
	t.Run("empty map", func(t *testing.T) {
		tagIDs := map[string]bool{}
		result := tag.TagIDsMapToSlice(tagIDs)
		gt.A(t, result).Length(0)
	})

	t.Run("valid tag IDs map", func(t *testing.T) {
		tagIDs := map[string]bool{
			"tag_1": true,
			"tag_2": true,
			"tag_3": true,
		}
		result := tag.TagIDsMapToSlice(tagIDs)
		gt.A(t, result).Length(3)
		// Check that all tag IDs are present (order doesn't matter)
		containsTag1 := false
		containsTag2 := false
		containsTag3 := false
		for _, id := range result {
			if id == "tag_1" {
				containsTag1 = true
			}
			if id == "tag_2" {
				containsTag2 = true
			}
			if id == "tag_3" {
				containsTag3 = true
			}
		}
		gt.Value(t, containsTag1).Equal(true)
		gt.Value(t, containsTag2).Equal(true)
		gt.Value(t, containsTag3).Equal(true)
	})
}
