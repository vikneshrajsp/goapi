package kafka

import (
	"strings"

	"github.com/segmentio/kafka-go"
)

// kafkaHeaderCarrier implements propagation.TextMapCarrier for kafka message
// headers so W3C tracecontext (traceparent/tracestate/baggage) crosses the broker.
type kafkaHeaderCarrier struct {
	headers *[]kafka.Header
}

func newKafkaHeaderCarrier(headers *[]kafka.Header) kafkaHeaderCarrier {
	return kafkaHeaderCarrier{headers: headers}
}

func (c kafkaHeaderCarrier) Get(key string) string {
	if c.headers == nil {
		return ""
	}
	for _, h := range *c.headers {
		if strings.EqualFold(h.Key, key) {
			return string(h.Value)
		}
	}
	return ""
}

func (c kafkaHeaderCarrier) Set(key, value string) {
	if c.headers == nil {
		return
	}
	*c.headers = append(*c.headers, kafka.Header{Key: key, Value: []byte(value)})
}

func (c kafkaHeaderCarrier) Keys() []string {
	if c.headers == nil {
		return nil
	}
	out := make([]string, 0, len(*c.headers))
	for _, h := range *c.headers {
		out = append(out, h.Key)
	}
	return out
}
