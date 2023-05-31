// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package logs // import "go.opentelemetry.io/collector/receiver/otlpreceiver/internal/logs"

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
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.opentelemetry.io/collector/receiver"
	rerrors "go.opentelemetry.io/collector/receiver/otlpreceiver/internal/errors"
)

const (
	dataFormatProtobuf = "protobuf"
	receiverTransport  = "grpc"

	scopeName = "go.opentelemetry.io/collector/receiver/otlpreceiver"
)

// Receiver is the type used to handle logs from OpenTelemetry exporters.
type Receiver struct {
	plogotlp.UnimplementedGRPCServer
	nextConsumer consumer.Logs
	obsrecv      *obsreport.Receiver

	requestDurationHistogram metric.Int64Histogram
}

// New creates a new Receiver reference.
func New(nextConsumer consumer.Logs, set receiver.CreateSettings, obsrecv *obsreport.Receiver) (*Receiver, error) {
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

// Export implements the service Export logs func.
func (r *Receiver) Export(ctx context.Context, req plogotlp.ExportRequest) (plogotlp.ExportResponse, error) {
	t0 := time.Now()
	var err error
	defer r.recordRequestDuration(ctx, t0, err)

	ld := req.Logs()
	numSpans := ld.LogRecordCount()
	if numSpans == 0 {
		return plogotlp.NewExportResponse(), nil
	}

	ctx = r.obsrecv.StartLogsOp(ctx)
	err = r.nextConsumer.ConsumeLogs(ctx, ld)
	r.obsrecv.EndLogsOp(ctx, dataFormatProtobuf, numSpans, err)

	return plogotlp.NewExportResponse(), err
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
			semconv.RPCServiceKey.String("log"),
			semconv.RPCMethodKey.String("export"),
			semconv.RPCSystemGRPC,
			semconv.RPCGRPCStatusCodeKey.Int(int(s.Code())),
		),
	)
}
