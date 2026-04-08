package opentelemetry

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/trace"
)

func SetupOTelSDK(ctx context.Context) (func(context.Context) error, error) {
	var shutdownFuncs []func(context.Context) error
	var err error

	// shutdown 会调用所有注册的清理函数。
	// 所有返回的错误都会被合并到一起。
	// 每个注册的清理函数仅会被调用一次。
	shutdown := func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	// handleErr 用户调用 shutdown 并合并返回的错误。
	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	// 设置 TracerProvider
	tp, err := newTracerProvider()
	if err != nil {
		handleErr(err)
		return shutdown, err
	}
	shutdownFuncs = append(shutdownFuncs, tp.Shutdown)
	otel.SetTracerProvider(tp)

	return shutdown, err

}

func newTracerProvider() (*trace.TracerProvider, error) {
	traceExporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
	)
	if err != nil {
		return nil, err
	}
	tp := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter),
	)
	return tp, nil
}

// func newMeterProvider() (*metric.MeterProvider, error) {
// 	metricExporter, err := stdoutmetric.New()
// 	if err != nil {
// 		return nil, err
// 	}
// 	mp := metric.NewMeterProvider(
// 		metric.WithReader(metric.NewPeriodicReader(metricExporter,
// 			metric.WithInterval(time.Minute),
// 		),
// 		),
// 	)
// 	return mp, nil
// }
