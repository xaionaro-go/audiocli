package tinymix

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/facebookincubator/go-belt/tool/logger"
)

type TinyMix struct {
	config *config
}

func New(ctx context.Context, opts ...Option) (*TinyMix, error) {
	cfg, err := options(opts).PreConfig().Build(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to build a config: %w", err)
	}
	c := &TinyMix{
		config: &cfg,
	}
	if cfg.TinymixType == TinymixTypeAuto {
		if err := c.detectTinymixType(ctx); err != nil {
			return nil, fmt.Errorf("unable to detect the tinymix binary type: %w", err)
		}
	}
	return c, nil
}

type ControlValue struct {
	Type     ValueType
	Num      int
	Name     string
	Values   []string
	Selected *int `json:",omitempty"`
}

func (c *TinyMix) detectTinymixType(ctx context.Context) error {
	stdout, _, err := c.config.ExecFunc(
		ctx,
		c.config.TinymixPath,
		nil,
		nil,
	)
	if err != nil {
		return fmt.Errorf("unable to execute tinymix binary: %w", err)
	}

	if strings.Contains(string(stdout), "get NAME|ID") {
		c.config.TinymixType = TinymixTypeUpstream
	} else {
		c.config.TinymixType = TinymixTypeGoogle
	}
	return nil
}

func (c *TinyMix) GetState(ctx context.Context) ([]ControlValue, error) {
	if c.config.DeviceIdx < 0 || c.config.DeviceName != "" {
		return nil, fmt.Errorf("selecting a device by name is not supported, yet")
	}
	stdout, stderr, err := c.config.ExecFunc(
		ctx,
		c.config.TinymixPath,
		[]string{"-t", "-D", fmt.Sprint(c.config.DeviceIdx), "-a"},
		nil,
	)
	if len(stderr) != 0 {
		return nil, fmt.Errorf("called tinymix: stderr was not empty: %s", stderr)
	}
	if err != nil {
		return nil, fmt.Errorf("calling tinymix error: %w", err)
	}

	r := bufio.NewReader(bytes.NewReader(stdout))

	_, _, err = r.ReadLine()
	if err != nil {
		return nil, fmt.Errorf("unable to get the first line from '%s'", stdout)
	}
	// TODO: implement device name parser

	if c.config.TinymixType == TinymixTypeGoogle {
		// skip the second line
		_, _, err = r.ReadLine()
		if err != nil {
			return nil, fmt.Errorf("unable to get the second line from '%s'", stdout)
		}
	}

	thirdLine, _, err := r.ReadLine()
	if err != nil {
		return nil, fmt.Errorf("unable to get the third line from '%s'", stdout)
	}

	fieldNames := strings.Split(string(thirdLine), "\t")
	// TODO: use the field appropriately during parsing below
	_ = fieldNames

	if c.config.TinymixType != TinymixTypeGoogle {
		return nil, fmt.Errorf("currently only the Google's version of tinymix is supported")
	}
	var values []ControlValue
	for {
		line, isPrefix, err := r.ReadLine()
		if isPrefix {
			return nil, fmt.Errorf("line '%s...' is too long", line)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("unable to read a line: %w", err)
		}
		if len(line) == 0 {
			continue
		}

		words := strings.Split(string(line), "\t")

		if len(words) < 5 {
			return nil, fmt.Errorf("not enough columns in line '%s': received %d, but expected at least 5", line, len(words))
		}

		controlIdx, err := strconv.ParseUint(words[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("unable to parse the control value index '%s' in line '%s': %w", words[0], line, err)
		}
		if int(controlIdx) != len(values) {
			return nil, fmt.Errorf("unexpected control value index %d (expected: %d) in line '%s'", controlIdx, len(values), line)
		}

		valueType, err := ParseValueType(words[1])
		if err != nil {
			return nil, fmt.Errorf("unable to parse the value type '%s' in lines '%s': %w", words[1], line, err)
		}

		num, err := strconv.ParseUint(words[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("unable to parse the amount of values '%s' in line '%s': %w", words[2], line, err)
		}

		name := words[3]

		var selectedValue *int
		innerValues := make([]string, 0, num)
		for i := 0; i < int(num); i++ {
			word := words[4+i]

			var v string
			switch valueType {
			case ValueTypeByte, ValueTypeInt:
				base := 10
				if valueType == ValueTypeByte {
					base = 16
				}
				vInt64, err := strconv.ParseInt(strings.Trim(word, " "), base, 64)
				if err != nil {
					return nil, fmt.Errorf("unable to parse the value '%s' in line '%s': %w", word, line, err)
				}
				v = fmt.Sprint(vInt64)
			case ValueTypeBool:
				switch strings.Trim(strings.ToUpper(word), " \t\r\n") {
				case "OFF":
					v = "true"
				case "ON":
					v = "false"
				default:
					return nil, fmt.Errorf("unexpected boolean value '%s' in line '%s'", word, line)
				}
			case ValueTypeEnum:
				if word[0] == '>' {
					word = word[1:]
					selectedValue = ptr(i)
				}
				v = word
			default:
				return nil, fmt.Errorf("internal error: unexpected value type '%s' in line '%s'", valueType, line)
			}
			innerValues = append(innerValues, v)
		}

		values = append(values, ControlValue{
			Type:     valueType,
			Num:      int(num),
			Name:     name,
			Values:   innerValues,
			Selected: selectedValue,
		})
	}

	return values, nil
}

func ptr[T any](v T) *T {
	return &v
}

func (c *TinyMix) LoadState(ctx context.Context, state []ControlValue) error {
	if c.config.DeviceIdx < 0 || c.config.DeviceName != "" {
		return fmt.Errorf("selecting a device by name is not supported, yet")
	}
	if c.config.TinymixType != TinymixTypeGoogle {
		return fmt.Errorf("currently only the Google's version of tinymix is supported")
	}

	for _, v := range state {
		var out []string
		for _, in := range v.Values {
			var _out string
			switch v.Type {
			case ValueTypeBool:
				switch in {
				case "true":
					_out = "1"
				case "false":
					_out = "0"
				default:
					return fmt.Errorf("invalid boolean value '%s'", in)
				}
				out = append(out, _out)
			case ValueTypeInt:
				_out = fmt.Sprint(in)
				out = append(out, _out)
			case ValueTypeByte:
				out = append(out, in)
			}
		}
		switch v.Type {
		case ValueTypeEnum:
			if v.Selected == nil && len(v.Values) < 2 {
				continue
			}
			out = append(out, v.Values[*v.Selected])
		}
		args := append([]string{"-D", fmt.Sprint(c.config.DeviceIdx), v.Name, "--"}, out...)

		_, stderr, err := c.config.ExecFunc(
			ctx,
			c.config.TinymixPath,
			args,
			nil,
		)
		if len(stderr) != 0 {
			logger.Errorf(ctx, "called tinymix: stderr was not empty: %s", stderr)
		}
		if err != nil {
			logger.Errorf(ctx, "calling tinymix error: %v", err)
		}
	}

	return nil
}
