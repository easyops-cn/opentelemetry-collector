// Copyright 2020 The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kafkaexporter

import (
	"github.com/Shopify/sarama"
	"go.opentelemetry.io/collector/consumer/pdata"
	otlpmetric "go.opentelemetry.io/collector/internal/data/protogen/collector/metrics/v1"
	otlptrace "go.opentelemetry.io/collector/internal/data/protogen/collector/trace/v1"
	v1 "go.opentelemetry.io/collector/internal/data/protogen/trace/v1"
)

var _ TracesMarshaller = (*otlpTracesPbMarshaller)(nil)
var _ MetricsMarshaller = (*otlpMetricsPbMarshaller)(nil)

type otlpTracesPbMarshaller struct {
}

func (m *otlpTracesPbMarshaller) Encoding() string {
	return defaultEncoding
}

func (m *otlpTracesPbMarshaller) Marshal(traces pdata.Traces, topic string) ([]*sarama.ProducerMessage, error) {
	messages := make([]*sarama.ProducerMessage, traces.ResourceSpans().Len())
	for i := 0; i < traces.ResourceSpans().Len(); i++ {
		var key []byte
		if span := traces.ResourceSpans().At(i).InstrumentationLibrarySpans().At(0).Spans().At(0); !span.TraceID().IsEmpty() {
			key = []byte(span.TraceID().HexString())
		}
		resourceSpans := traces.ResourceSpans()
		request := otlptrace.ExportTraceServiceRequest{ResourceSpans: []*v1.ResourceSpans{pdata.TraceToOtlp(resourceSpans, i)}}
		bts, err := request.Marshal()
		if err != nil {
			return nil, err
		}
		messages[i] = &sarama.ProducerMessage{
			Value: sarama.ByteEncoder(bts),
			Topic: topic,
			Key:   sarama.ByteEncoder(key),
		}
	}

	return messages, nil
}

type otlpMetricsPbMarshaller struct {
}

func (m *otlpMetricsPbMarshaller) Encoding() string {
	return defaultEncoding
}

func (m *otlpMetricsPbMarshaller) Marshal(metrics pdata.Metrics) ([]Message, error) {
	request := otlpmetric.ExportMetricsServiceRequest{
		ResourceMetrics: pdata.MetricsToOtlp(metrics),
	}
	bts, err := request.Marshal()
	if err != nil {
		return nil, err
	}
	return []Message{{Value: bts}}, nil
}

type otlpLogsPbMarshaller struct {
}

func (m *otlpLogsPbMarshaller) Encoding() string {
	return defaultEncoding
}

func (m *otlpLogsPbMarshaller) Marshal(ld pdata.Logs) ([]Message, error) {
	bts, err := ld.ToOtlpProtoBytes()
	if err != nil {
		return nil, err
	}
	return []Message{{Value: bts}}, nil
}
