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

package service

import (
	"net/http"
	"strings"
	"unicode"

	"go.opencensus.io/stats/view"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/exporter/jaegerexporter"
	"go.opentelemetry.io/collector/internal/collector/telemetry"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/batchprocessor"
	fluentobserv "go.opentelemetry.io/collector/receiver/fluentforwardreceiver/observ"
	"go.opentelemetry.io/collector/receiver/kafkareceiver"
	telemetry2 "go.opentelemetry.io/collector/service/internal/telemetry"
)

// applicationTelemetry is application's own telemetry.
var applicationTelemetry appTelemetryExporter = &appTelemetry{}

type appTelemetryExporter interface {
	init(asyncErrorChannel chan<- error, ballastSizeBytes uint64, logger *zap.Logger) error
	shutdown() error
}

type appTelemetry struct {
	views  []*view.View
	server *http.Server
}

func (tel *appTelemetry) init(asyncErrorChannel chan<- error, ballastSizeBytes uint64, logger *zap.Logger) error {
	level := configtelemetry.GetMetricsLevelFlagValue()
	metricsAddr := telemetry.GetMetricsAddr()

	if level == configtelemetry.LevelNone || metricsAddr == "" {
		return nil
	}

	processMetricsViews, err := telemetry2.NewProcessMetricsViews(ballastSizeBytes)
	if err != nil {
		return err
	}

	var views []*view.View
	views = append(views, batchprocessor.MetricViews()...)
	views = append(views, fluentobserv.MetricViews()...)
	views = append(views, jaegerexporter.MetricViews()...)
	views = append(views, kafkareceiver.MetricViews()...)
	views = append(views, obsreport.Configure(level)...)
	views = append(views, processMetricsViews.Views()...)
	views = append(views, processor.MetricViews()...)

	tel.views = views
	// 报错关键字；a different view with the same name is already registered
	// 启用多个协程的时候会多次注册公共view，导致报错，暂时屏蔽该报错
	// 以后有时间再修 TODO
	if err := view.Register(views...); err != nil && !strings.Contains(err.Error(), "a different view with the same name is already registered") {
		return err
	}

	processMetricsViews.StartCollection()

	// 因为启动多个application,这里可能会重复监听prometheus端口
	// TODO 以后有时间处理
	// Until we can use a generic metrics exporter, default to Prometheus.
	//opts := prometheus.Options{
	//	Namespace: telemetry.GetMetricsPrefix(),
	//}
	//
	//var instanceID string
	//if telemetry.GetAddInstanceID() {
	//	instanceUUID, _ := uuid.NewRandom()
	//	instanceID = instanceUUID.String()
	//	opts.ConstLabels = map[string]string{
	//		sanitizePrometheusKey(conventions.AttributeServiceInstance): instanceID,
	//	}
	//}

	//pe, err := prometheus.NewExporter(opts)
	//if err != nil {
	//	return err
	//}

	//view.RegisterExporter(pe)

	//logger.Info(
	//	"Serving Prometheus metrics",
	//	zap.String("address", metricsAddr),
	//	zap.Int8("level", int8(level)), // TODO: make it human friendly
	//	zap.String(conventions.AttributeServiceInstance, instanceID),
	//)

	//mux := http.NewServeMux()
	//mux.Handle("/metrics", pe)

	//tel.server = &http.Server{
	//	Addr:    metricsAddr,
	//	Handler: mux,
	//}

	//go func() {
	//	serveErr := tel.server.ListenAndServe()
	//	if serveErr != nil && serveErr != http.ErrServerClosed {
	//		asyncErrorChannel <- serveErr
	//	}
	//}()

	return nil
}

func (tel *appTelemetry) shutdown() error {
	view.Unregister(tel.views...)

	if tel.server != nil {
		return tel.server.Close()
	}

	return nil
}

func sanitizePrometheusKey(str string) string {
	runeFilterMap := func(r rune) rune {
		if unicode.IsDigit(r) || unicode.IsLetter(r) || r == '_' {
			return r
		}
		return '_'
	}
	return strings.Map(runeFilterMap, str)
}
