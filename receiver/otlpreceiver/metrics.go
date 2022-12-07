package otlpreceiver

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
)

const (
	formatKey = "format"
)

var (
	TagKeyFormat, _ = tag.NewKey(formatKey)
	TagKeyRoute, _  = tag.NewKey("route")

	statsRequestDuration = stats.Int64(
		obsmetrics.ReceiverPrefix+"request_duration",
		"The duration of a request to the pipeline.",
		stats.UnitMilliseconds)
)

func newInt64View(m *stats.Int64Measure, a *view.Aggregation, t ...tag.Key) *view.View {
	return &view.View{
		Name:        m.Name(),
		Measure:     m,
		Description: m.Description(),
		Aggregation: a,
		TagKeys:     t,
	}
}

func views(level configtelemetry.Level) []*view.View {
	views := []*view.View{}
	switch level {
	case configtelemetry.LevelDetailed:
		fallthrough

	case configtelemetry.LevelNormal:
		fallthrough

	case configtelemetry.LevelBasic:
		views = append(views, []*view.View{
			newInt64View(statsRequestDuration,
				view.Distribution(0, 5, 10, 25, 50, 75, 100, 500, 1e3, 2e3, 3e3, 4e3, 5e3, 6e3, 7e3, 8e3, 9e3, 10e3, 15e3),
				obsmetrics.TagKeyTransport, TagKeyFormat, TagKeyRoute),
		}...)

	case configtelemetry.LevelNone:
	default:
	}
	return views
}
