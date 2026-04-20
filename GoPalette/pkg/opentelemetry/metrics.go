package opentelemetry

import (
	"github.com/go-kratos/kratos/v2/middleware/metrics"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

type ServiceName string

// NewRequestCounter 创建请求计数器
func NewRequestCounter(name ServiceName) metric.Int64Counter {
	meter := otel.Meter(string(name))
	counter, err := metrics.DefaultRequestsCounter(meter, metrics.DefaultServerRequestsCounterName)
	if err != nil {
		panic(err)
	}
	return counter
}

// NewSecondsHistogram 创建耗时直方图
func NewSecondsHistogram(name ServiceName) metric.Float64Histogram {
	meter := otel.Meter(string(name))
	histogram, err := metrics.DefaultSecondsHistogram(meter, metrics.DefaultServerSecondsHistogramName)
	if err != nil {
		panic(err)
	}
	return histogram
}
