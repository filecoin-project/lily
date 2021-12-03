package metrics

import (
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

var log = logging.Logger("metrics")

// NewJaegerTraceProvider returns a new and configured TracerProvider backed by Jaeger.
func NewJaegerTraceProvider(serviceName, agentEndpoint string, sampleRatio float64) (*tracesdk.TracerProvider, error) {
	log.Infow("creating jaeger trace provider", "serviceName", serviceName, "ratio", sampleRatio, "endpoint", agentEndpoint)
	var sampler tracesdk.Sampler
	if sampleRatio < 1 && sampleRatio > 0 {
		sampler = tracesdk.ParentBased(tracesdk.TraceIDRatioBased(sampleRatio))
	} else if sampleRatio == 1 {
		sampler = tracesdk.AlwaysSample()
	} else {
		sampler = tracesdk.NeverSample()
	}

	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(agentEndpoint)))
	if err != nil {
		return nil, err
	}
	tp := tracesdk.NewTracerProvider(
		// Always be sure to batch in production.
		tracesdk.WithBatcher(exp),
		// Use the provided sampling ratio.
		tracesdk.WithSampler(sampler),
		// Record information about this application in an Resource.
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
		)),
	)
	return tp, nil
}
