package tracing

import (
	"context"
	"io"
	"time"

	"github.com/Staspol216/gh1/pkg/logger"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/uber/jaeger-client-go/config"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func UnaryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (resp interface{}, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, info.FullMethod)
	defer span.Finish()

	resp, err = handler(ctx, req)

	if err != nil {
		RecordError(span, err)
	}

	return resp, err
}

func StartSpanFromContext(ctx context.Context, operationName string) (opentracing.Span, context.Context) {
	return opentracing.StartSpanFromContext(ctx, operationName)
}

func FinishSpan(span opentracing.Span, startTime time.Time, err error) {
	span.SetTag("duration_ms", time.Since(startTime).Milliseconds())
	if err != nil {
		RecordError(span, err)
	}
	span.Finish()
}

func RecordError(span opentracing.Span, err error) {
	ext.Error.Set(span, true)
	span.LogKV("error", err.Error())
}

func InitTracer(serviceName string, collectorEndpoint string) (opentracing.Tracer, io.Closer) {
	cfg := config.Configuration{
		ServiceName: serviceName,
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans:          true,
			CollectorEndpoint: collectorEndpoint,
		},
	}

	// config.Logger(jaeger.StdLogger) - debug
	tracer, closer, err := cfg.NewTracer()
	if err != nil {
		app_logger.MyLogger.Fatal("failed to initialize tracer", zap.Error(err))
	}
	opentracing.SetGlobalTracer(tracer)

	app_logger.MyLogger.Info("tracer initialized", zap.String("collector_endpoint", collectorEndpoint))

	return tracer, closer
}
