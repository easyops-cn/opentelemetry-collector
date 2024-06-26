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
	"bytes"

	"github.com/Shopify/sarama"
	"github.com/gogo/protobuf/jsonpb"
	jaegerproto "github.com/jaegertracing/jaeger/model"

	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/pdata"
	jaegertranslator "go.opentelemetry.io/collector/translator/trace/jaeger"
)

type jaegerMarshaller struct {
	marshaller jaegerSpanMarshaller
}

var _ TracesMarshaller = (*jaegerMarshaller)(nil)

func (j jaegerMarshaller) Marshal(traces pdata.Traces, topic string) ([]*sarama.ProducerMessage, error) {
	batches, err := jaegertranslator.InternalTracesToJaegerProto(traces)
	if err != nil {
		return nil, err
	}
	var messages []*sarama.ProducerMessage
	var errs []error
	for _, batch := range batches {
		for _, span := range batch.Spans {
			span.Process = batch.Process
			bts, err := j.marshaller.marshall(span)
			// continue to process spans that can be serialized
			if err != nil {
				errs = append(errs, err)
				continue
			}
			key := []byte(span.TraceID.String())
			messages = append(messages, &sarama.ProducerMessage{
				Value: sarama.ByteEncoder(bts),
				Topic: topic,
				Key:   sarama.ByteEncoder(key),
			})
		}
	}
	return messages, consumererror.CombineErrors(errs)
}

func (j jaegerMarshaller) Encoding() string {
	return j.marshaller.encoding()
}

type jaegerSpanMarshaller interface {
	marshall(span *jaegerproto.Span) ([]byte, error)
	encoding() string
}

type jaegerProtoSpanMarshaller struct {
}

var _ jaegerSpanMarshaller = (*jaegerProtoSpanMarshaller)(nil)

func (p jaegerProtoSpanMarshaller) marshall(span *jaegerproto.Span) ([]byte, error) {
	return span.Marshal()
}

func (p jaegerProtoSpanMarshaller) encoding() string {
	return "jaeger_proto"
}

type jaegerJSONSpanMarshaller struct {
	pbMarshaller *jsonpb.Marshaler
}

var _ jaegerSpanMarshaller = (*jaegerJSONSpanMarshaller)(nil)

func newJaegerJSONMarshaller() *jaegerJSONSpanMarshaller {
	return &jaegerJSONSpanMarshaller{
		pbMarshaller: &jsonpb.Marshaler{},
	}
}

func (p jaegerJSONSpanMarshaller) marshall(span *jaegerproto.Span) ([]byte, error) {
	out := new(bytes.Buffer)
	err := p.pbMarshaller.Marshal(out, span)
	return out.Bytes(), err
}

func (p jaegerJSONSpanMarshaller) encoding() string {
	return "jaeger_json"
}
