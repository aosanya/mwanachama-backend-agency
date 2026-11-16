package models

import "encoding/json"

type carriedKeys struct {
	keys map[string]bool
}

func (c *carriedKeys) record(data []byte) error {
	var present map[string]json.RawMessage
	if err := json.Unmarshal(data, &present); err != nil {
		return err
	}
	c.keys = make(map[string]bool, len(present))
	for key := range present {
		c.keys[key] = true
	}
	return nil
}

func (c carriedKeys) Carries(field string) bool {
	return c.keys == nil || c.keys[field]
}

func (c carriedKeys) FromRequest() bool {
	return c.keys != nil
}
