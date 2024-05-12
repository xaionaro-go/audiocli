package tinymix

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ValueType int

var _ json.Marshaler = (ValueType)(0)
var _ json.Unmarshaler = (*ValueType)(nil)

const (
	ValueTypeUndefined = ValueType(iota)
	ValueTypeBool
	ValueTypeEnum
	ValueTypeInt
	ValueTypeByte
	EndOfValueType
)

func ParseValueType(in string) (ValueType, error) {
	in = strings.Trim(strings.ToUpper(in), " \t\n\r")
	for vt := ValueType(1); vt < EndOfValueType; vt++ {
		if in == vt.String() {
			return vt, nil
		}
	}
	return ValueTypeUndefined, fmt.Errorf("unknown value type: '%s'", in)
}

func (vt ValueType) String() string {
	switch vt {
	case ValueTypeUndefined:
		return "<undefined>"
	case ValueTypeBool:
		return "BOOL"
	case ValueTypeEnum:
		return "ENUM"
	case ValueTypeInt:
		return "INT"
	case ValueTypeByte:
		return "BYTE"
	default:
		return fmt.Sprintf("<unknown_ValueType_%d>", int(vt))
	}
}

func (vt ValueType) MarshalJSON() ([]byte, error) {
	return []byte(`"` + vt.String() + `"`), nil
}

func (vt *ValueType) UnmarshalJSON(b []byte) error {
	if len(b) < 2 || b[0] != '"' || b[len(b)-1] != '"' {
		return fmt.Errorf("invalid JSON string")
	}
	b = b[1 : len(b)-1]
	p, err := ParseValueType(string(b))
	if err != nil {
		return err
	}
	*vt = p
	return nil
}
