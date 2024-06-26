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

package data

import (
	"encoding/hex"
	"errors"
)

const traceIDSize = 16

var errInvalidTraceIDSize = errors.New("invalid length for SpanID")

// TraceID is a custom data type that is used for all trace_id fields in OTLP
// Protobuf messages.
type TraceID struct {
	id         [traceIDSize]byte
	idSlice    []byte // for unlimited traceId size
	useIdSlice bool
}

// NewTraceID creates a TraceID from a byte slice.
func NewTraceID(bytes [16]byte) TraceID {
	return TraceID{
		id: bytes,
	}
}

// NewTraceID creates a TraceID from a byte slice.
func NewTraceIDWithUnlimitedSize(bytes []byte) TraceID {
	return TraceID{
		idSlice:    bytes,
		useIdSlice: true,
	}
}

// HexString returns hex representation of the ID.
func (tid TraceID) HexString() string {
	if tid.IsEmpty() {
		return ""
	}
	if tid.useIdSlice {
		return string(tid.idSlice)
	} else {
		return hex.EncodeToString(tid.id[:])
	}
}

// Size returns the size of the data to serialize.
func (tid *TraceID) Size() int {
	if tid.IsEmpty() {
		return 0
	}
	if tid.useIdSlice {
		return len(tid.idSlice)
	} else {
		return traceIDSize
	}
}

// Equal returns true if ids are equal.
func (tid TraceID) Equal(that TraceID) bool {
	if tid.useIdSlice {
		if len(tid.idSlice) != len(that.idSlice) {
			return false
		}
		for i, n := range tid.idSlice {
			if n != that.idSlice[i] {
				return false
			}
		}
		return true
	} else {
		return tid.id == that.id
	}
}

// IsValid returns true if id contains at leas one non-zero byte.
func (tid TraceID) IsEmpty() bool {
	if tid.useIdSlice {
		return len(tid.idSlice) == 0
	} else {
		return tid.id == [16]byte{}
	}
}

// Bytes returns the byte array representation of the TraceID.
func (tid TraceID) Bytes() [16]byte {
	if tid.useIdSlice {
		panic("could not use id slice")
	}
	return tid.id
}

// MarshalTo converts trace ID into a binary representation. Called by Protobuf serialization.
func (tid *TraceID) MarshalTo(data []byte) (n int, err error) {
	if tid.IsEmpty() {
		return 0, nil
	}
	if tid.useIdSlice {
		n, err = marshalBytes(data, tid.idSlice[:])
		return n, err
	} else {
		return marshalBytes(data, tid.id[:])
	}
}

// Unmarshal inflates this trace ID from binary representation. Called by Protobuf serialization.
func (tid *TraceID) Unmarshal(data []byte) error {
	if len(data) == 0 {
		tid.id = [16]byte{}
		return nil
	}

	if len(data) != traceIDSize {
		tid.useIdSlice = true
		tid.idSlice = make([]byte, len(data))
		copy(tid.idSlice, data)
	} else {
		copy(tid.id[:], data)
	}
	return nil
}

// MarshalJSON converts trace id into a hex string enclosed in quotes.
func (tid TraceID) MarshalJSON() ([]byte, error) {
	if tid.IsEmpty() {
		return []byte(`""`), nil
	}
	if tid.useIdSlice {
		return marshalJSON(tid.idSlice)
	} else {
		return marshalJSON(tid.id[:])
	}
}

// UnmarshalJSON inflates trace id from hex string, possibly enclosed in quotes.
// Called by Protobuf JSON deserialization.
func (tid *TraceID) UnmarshalJSON(data []byte) error {
	if len(data) != traceIDSize {
		tid.useIdSlice = true
		tid.idSlice = make([]byte, len(data))
		return unmarshalJSON(tid.idSlice[:], data)
	} else {
		tid.id = [16]byte{}
		return unmarshalJSON(tid.id[:], data)
	}
}
