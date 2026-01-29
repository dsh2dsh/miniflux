package model

import (
	"encoding/json"
	"fmt"
)

type anyObject struct {
	value any
	raw   json.RawMessage
}

var (
	_ json.Marshaler   = (*anyObject)(nil)
	_ json.Unmarshaler = (*anyObject)(nil)
)

func newAnyObject(data any) *anyObject {
	return &anyObject{value: data}
}

func (self *anyObject) MarshalJSON() ([]byte, error) {
	if self.value != nil {
		b, err := json.Marshal(self.value)
		if err != nil {
			return nil, fmt.Errorf("marshal anyObject value: %w", err)
		}
		return b, nil
	}

	if self.raw != nil {
		b, err := self.raw.MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("marshal anyObject raw: %w", err)
		}
		return b, nil
	}
	return nil, nil
}

func (self *anyObject) UnmarshalJSON(b []byte) error {
	self.value = nil
	if err := json.Unmarshal(b, &self.raw); err != nil {
		return fmt.Errorf("unmarshal anyObject: %w", err)
	}
	return nil
}

func (self *anyObject) Decode(v any) error {
	if self.raw == nil {
		return nil
	}

	if err := json.Unmarshal(self.raw, v); err != nil {
		return fmt.Errorf("unmarshal anyObject to %T: %w", v, err)
	}
	return nil
}
