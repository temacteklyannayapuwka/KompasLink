package ui

import (
	"github.com/lxn/walk"
	"kompaslink/internal/diagnostic"
)

type referenceModel struct {
	walk.TableModelBase
	rows []diagnostic.Reference
}

func (m *referenceModel) RowCount() int { return len(m.rows) }

func (m *referenceModel) Value(row, column int) any {
	if row < 0 || row >= len(m.rows) {
		return ""
	}
	r := m.rows[row]
	switch column {
	case 0:
		return r.Path
	case 1:
		return r.Status
	case 2:
		return r.Type
	case 3:
		return r.Detail
	}
	return ""
}

func (m *referenceModel) set(rows []diagnostic.Reference) {
	m.rows = rows
	m.PublishRowsReset()
}
