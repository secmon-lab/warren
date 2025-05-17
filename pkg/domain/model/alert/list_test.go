package alert

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/test"
)

func TestList_FillMetadata(t *testing.T) {
	client := test.NewGeminiClient(t)

	list := NewList(t.Context(), slack.Thread{}, &slack.User{}, Alerts{
		{
			ID: types.NewAlertID(),
			Metadata: Metadata{
				Title:       "Test Alert",
				Description: "Test Description",
				Attributes: []Attribute{
					{
						Key:   "Test Attribute",
						Value: "Test Value",
					},
					{
						Key:   "customer_id",
						Value: "1234567890",
					},
				},
			},
		},
		{
			ID: types.NewAlertID(),
			Metadata: Metadata{
				Title:       "Test Alert 2",
				Description: "Test Description 2",
				Attributes: []Attribute{
					{
						Key:   "Test Attribute 2",
						Value: "Test Value 2",
					},
					{
						Key:   "customer_id",
						Value: "1234567890",
					},
				},
			},
		},
	})

	err := list.FillMetadata(t.Context(), client)
	if err != nil {
		t.Fatalf("failed to fill metadata: %v", err)
	}

	gt.NotEqual(t, list.Metadata.Title, "")
	gt.NotEqual(t, list.Metadata.Description, "")
	t.Logf("metadata: %+v", list.Metadata)
}
