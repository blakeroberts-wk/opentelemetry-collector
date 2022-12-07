package otlpreceiver

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/obsreport"
)

const (
	transportKey = "transport"
	formatKey    = "format"

	// AcceptedSpansKey used to identify spans accepted by the Collector.
	AcceptedSpansKey = "accepted_spans"
	// RefusedSpansKey used to identify spans refused (ie.: not ingested) by the Collector.
	RefusedSpansKey = "refused_spans"

	// AcceptedMetricPointsKey used to identify metric points accepted by the Collector.
	AcceptedMetricPointsKey = "accepted_metric_points"
	// RefusedMetricPointsKey used to identify metric points refused (ie.: not ingested) by the
	// Collector.
	RefusedMetricPointsKey = "refused_metric_points"

	// AcceptedLogRecordsKey used to identify log records accepted by the Collector.
	AcceptedLogRecordsKey = "accepted_log_records"
	// RefusedLogRecordsKey used to identify log records refused (ie.: not ingested) by the
	// Collector.
	RefusedLogRecordsKey = "refused_log_records"
)

var (
	TagKeyTransport, _ = tag.NewKey(transportKey)
	TagFormatKey, _    = tag.NewKey(formatKey)

	statsRequestDuration = stats.Int64(
		"request_duration",
		"The duration of a request to the pipeline.",
		stats.UnitMilliseconds)
)

func newInt64View(m *stats.Int64Measure, a *view.Aggregation, t ...tag.Key) *view.View {
	name := obsreport.BuildProcessorCustomMetricName(typeStr, m.Name())
	return &view.View{
		Name:        name,
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
				view.Distribution(0, 5, 10, 25, 50, 75, 100, 250, 500, 1e3, 5e3, 1e4),
				TagKeyTransport, TagFormatKey),
		}...)

	case configtelemetry.LevelNone:
	default:
	}
	return views
}
