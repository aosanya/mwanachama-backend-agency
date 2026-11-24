package models

import "encoding/json"

const (
	RecordFieldRoleID       = "role_id"
	RecordFieldEditRoleID   = "edit_role_id"
	RecordFieldViewerRoleID = "viewer_role_id"
	RecordFieldDeleteRoleID = "delete_role_id"
)

func GoneRecordRoleFields() []string {
	return []string{RecordFieldRoleID, RecordFieldEditRoleID, RecordFieldViewerRoleID, RecordFieldDeleteRoleID}
}

func goneRoleFieldsIn(data []byte) ([]string, error) {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, err
	}
	var carried []string
	for _, name := range GoneRecordRoleFields() {
		raw, found := probe[name]
		if !found {
			continue
		}
		var value string
		if json.Unmarshal(raw, &value) == nil && value == "" {
			continue
		}
		carried = append(carried, name)
	}
	return carried, nil
}

func (r *Record) UnmarshalJSON(data []byte) error {
	carried, err := goneRoleFieldsIn(data)
	if err != nil {
		return err
	}
	type plain Record
	var read plain
	if err := json.Unmarshal(data, &read); err != nil {
		return err
	}
	*r = Record(read)
	r.carriedRoleFields = carried
	return nil
}

func (r Record) CarriedGoneRoleFields() []string {
	return r.carriedRoleFields
}

func (r *DocumentRecord) UnmarshalJSON(data []byte) error {
	carried, err := goneRoleFieldsIn(data)
	if err != nil {
		return err
	}
	type plain DocumentRecord
	var read plain
	if err := json.Unmarshal(data, &read); err != nil {
		return err
	}
	*r = DocumentRecord(read)
	r.carriedRoleFields = carried
	return nil
}

func (r DocumentRecord) CarriedGoneRoleFields() []string {
	return r.carriedRoleFields
}
