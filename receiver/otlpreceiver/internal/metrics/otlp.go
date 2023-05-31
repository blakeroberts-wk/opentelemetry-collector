// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package metrics // import "go.opentelemetry.io/collector/receiver/otlpreceiver/internal/metrics"

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.opentelemetry.io/collector/receiver"
	rerrors "go.opentelemetry.io/collector/receiver/otlpreceiver/internal/errors"
)

const (
	dataFormatProtobuf = "protobuf"
	receiverTransport  = "grpc"

	scopeName = "go.opentelemetry.io/collector/receiver/otlpreceiver"
)

// Receiver is the type used to handle metrics from OpenTelemetry exporters.
type Receiver struct {
	pmetricotlp.UnimplementedGRPCServer
	nextConsumer consumer.Metrics
	obsrecv      *obsreport.Receiver

	requestDurationHistogram metric.Int64Histogram
}

// New creates a new Receiver reference.
func New(nextConsumer consumer.Metrics, set receiver.CreateSettings, obsrecv *obsreport.Receiver) (*Receiver, error) {
	r := &Receiver{
		nextConsumer: nextConsumer,
		obsrecv:      obsrecv,
	}

	var err error
	r.requestDurationHistogram, err = set.MeterProvider.Meter(scopeName).Int64Histogram(
		"rpc.server.duration",
		metric.WithUnit("ms"))

	return r, err
}

// Export implements the service Export metrics func.
func (r *Receiver) Export(ctx context.Context, req pmetricotlp.ExportRequest) (pmetricotlp.ExportResponse, error) {
	t0 := time.Now()
	var err error
	defer r.recordRequestDuration(ctx, t0, err)

	md := req.Metrics()
	dataPointCount := md.DataPointCount()
	if dataPointCount == 0 {
		return pmetricotlp.NewExportResponse(), nil
	}

	ctx = r.obsrecv.StartMetricsOp(ctx)
	err = r.nextConsumer.ConsumeMetrics(ctx, md)
	r.obsrecv.EndMetricsOp(ctx, dataFormatProtobuf, dataPointCount, err)

	return pmetricotlp.NewExportResponse(), err
}

func (r *Receiver) recordRequestDuration(ctx context.Context, t0 time.Time, err error) {
	duration := time.Since(t0).Milliseconds()
	s, ok := status.FromError(err)
	if !ok && err != nil {
		if errors.Is(err, &rerrors.ErrorRateLimited{}) {
			s = status.New(codes.Unavailable, err.Error())
		} else {
			s = status.New(codes.Unknown, err.Error())
		}
	}
	r.requestDurationHistogram.Record(ctx, duration,
		metric.WithAttributes(
			semconv.RPCServiceKey.String("metric"),
			semconv.RPCMethodKey.String("export"),
			semconv.RPCSystemGRPC,
			semconv.RPCGRPCStatusCodeKey.Int(int(s.Code())),
		),
	)
}
