// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package trace // import "go.opentelemetry.io/collector/receiver/otlpreceiver/internal/trace"

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/instrument/syncint64"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	"go.opentelemetry.io/collector/receiver"
	rerrors "go.opentelemetry.io/collector/receiver/otlpreceiver/internal/errors"
)

const (
	dataFormatProtobuf = "protobuf"
	receiverTransport  = "grpc"

	scopeName = "go.opentelemetry.io/collector/receiver/otlpreceiver"
)

// Receiver is the type used to handle spans from OpenTelemetry exporters.
type Receiver struct {
	nextConsumer consumer.Traces
	obsrecv      *obsreport.Receiver

	requestDurationHistogram syncint64.Histogram
}

// New creates a new Receiver reference.
func New(nextConsumer consumer.Traces, set receiver.CreateSettings) (*Receiver, error) {
	obsrecv, err := obsreport.NewReceiver(obsreport.ReceiverSettings{
		ReceiverID:             set.ID,
		Transport:              receiverTransport,
		ReceiverCreateSettings: set,
	})
	if err != nil {
		return nil, err
	}

	r := &Receiver{
		nextConsumer: nextConsumer,
		obsrecv:      obsrecv,
	}

	r.requestDurationHistogram, err = set.MeterProvider.Meter(scopeName).SyncInt64().Histogram(
		"rpc.server.duration",
		instrument.WithUnit("ms"))

	return r, err
}

// Export implements the service Export traces func.
func (r *Receiver) Export(ctx context.Context, req ptraceotlp.ExportRequest) (ptraceotlp.ExportResponse, error) {
	t0 := time.Now()
	var err error
	defer r.recordRequestDuration(ctx, t0, err)

	td := req.Traces()
	// We need to ensure that it propagates the receiver name as a tag
	numSpans := td.SpanCount()
	if numSpans == 0 {
		return ptraceotlp.NewExportResponse(), nil
	}

	ctx = r.obsrecv.StartTracesOp(ctx)
	err = r.nextConsumer.ConsumeTraces(ctx, td)
	r.obsrecv.EndTracesOp(ctx, dataFormatProtobuf, numSpans, err)

	return ptraceotlp.NewExportResponse(), err
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
		semconv.RPCServiceKey.String("trace"),
		semconv.RPCMethodKey.String("export"),
		semconv.RPCSystemGRPC,
		semconv.RPCGRPCStatusCodeKey.Int(int(s.Code())))
}
