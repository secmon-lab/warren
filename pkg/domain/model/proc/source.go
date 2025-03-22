package proc

import (
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/source"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

func targetFromSource(target string, thread slack.Thread) source.Source {
	switch target {
	case "root":
		return source.RootAlertList(thread)
	case "last":
		return source.LatestAlertList(thread)
	default:
		return source.AlertListID(types.AlertListID(target))
	}
}
