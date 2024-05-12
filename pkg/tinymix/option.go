package tinymix

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"github.com/facebookincubator/go-belt/tool/logger"
)

var (
	DefaultTinymixPath = func(ctx context.Context) string {
		return "tinymix"
	}

	DefaultDeviceIdx = func(ctx context.Context) uint {
		return 0
	}

	DefaultExecFunc = func(ctx context.Context) ExecFunc {
		return func(
			ctx context.Context,
			name string,
			args []string,
			stdin []byte,
		) ([]byte, []byte, error) {
			logger.Debugf(ctx, "running %s with args %v", name, args)
			cmd := exec.Command(name, args...)
			cmd.Stdin = bytes.NewReader(stdin)
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			err := cmd.Run()
			logger.Debugf(ctx, "running result of %s with args %v: e:%v", name, args, err)
			return stdout.Bytes(), stderr.Bytes(), err
		}
	}
)

type TinymixType int

const (
	TinymixTypeAuto = TinymixType(iota)
	TinymixTypeUpstream
	TinymixTypeGoogle
)

type preConfig struct {
	TinymixPath *string
	TinymixType TinymixType
	DeviceIdx   *uint
	DeviceName  *string
	ExecFunc    ExecFunc
}

type Option interface {
	apply(*preConfig)
}

type OptionTinymixPath string

func (opt OptionTinymixPath) apply(cfg *preConfig) {
	cfg.TinymixPath = (*string)(&opt)
}

type OptionDeviceIdx uint

func (opt OptionDeviceIdx) apply(cfg *preConfig) {
	cfg.DeviceIdx = (*uint)(&opt)
}

type OptionDeviceName string

func (opt OptionDeviceName) apply(cfg *preConfig) {
	cfg.DeviceName = (*string)(&opt)
}

type ExecFunc func(
	ctx context.Context,
	name string,
	args []string,
	stdin []byte,
) (stdout, stderr []byte, err error)

type OptionExec ExecFunc

func (opt OptionExec) apply(cfg *preConfig) {
	cfg.ExecFunc = ExecFunc(opt)
}

type options []Option

func (s options) PreConfig() preConfig {
	c := preConfig{}
	for _, o := range s {
		o.apply(&c)
	}
	return c
}

type config struct {
	TinymixPath string
	TinymixType TinymixType
	DeviceIdx   int
	DeviceName  string
	ExecFunc    ExecFunc
}

func (preCfg preConfig) Build(ctx context.Context) (config, error) {
	cfg := config{
		TinymixPath: DefaultTinymixPath(ctx),
		TinymixType: preCfg.TinymixType,
		DeviceIdx:   int(DefaultDeviceIdx(ctx)),
		ExecFunc:    DefaultExecFunc(ctx),
	}

	if preCfg.TinymixPath != nil {
		cfg.TinymixPath = *preCfg.TinymixPath
	}

	if preCfg.DeviceIdx != nil && preCfg.DeviceName != nil {
		return config{}, fmt.Errorf("options DeviceIdx and DeviceName cannot be used together")
	}

	switch {
	case preCfg.DeviceName != nil:
		cfg.DeviceIdx = -1
		cfg.DeviceName = *preCfg.DeviceName
	case preCfg.DeviceIdx != nil:
		cfg.DeviceIdx = int(*preCfg.DeviceIdx)
	}

	if preCfg.ExecFunc != nil {
		cfg.ExecFunc = preCfg.ExecFunc
	}

	return cfg, nil
}
