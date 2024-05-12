package commands

import (
	"context"
	"encoding/json"
	"os"

	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/spf13/cobra"
	"github.com/xaionaro-go/audiocli/pkg/tinymix"
)

var (
	// Access these variables only from a main package:

	Root = &cobra.Command{
		Use: os.Args[0],
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()
			l := logger.FromCtx(ctx).WithLevel(LoggerLevel)
			ctx = logger.CtxWithLogger(ctx, l)
			cmd.SetContext(ctx)
			logger.Debugf(ctx, "log-level: %v", LoggerLevel)
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()
			logger.Debug(ctx, "end")
		},
	}

	Mixer = &cobra.Command{
		Use: "mixer",
	}

	MixerSave = &cobra.Command{
		Use:  "save <file>",
		Args: cobra.ExactArgs(1),
		Run:  mixerSave,
	}

	MixerLoad = &cobra.Command{
		Use:  "load <file>",
		Args: cobra.ExactArgs(1),
		Run:  mixerLoad,
	}

	LoggerLevel = logger.LevelWarning
)

func init() {
	Root.PersistentFlags().Var(&LoggerLevel, "log-level", "logging level")
	Root.AddCommand(Mixer)
	Mixer.PersistentFlags().String("tinymix-path", "", "the path to the tinymix tool")
	Mixer.PersistentFlags().Int("device-idx", -1, "the index of the device (cannot be used with --device-name)")
	Mixer.PersistentFlags().String("device-name", "", "the name of the device (cannot be used with --device-idx)")
	Mixer.AddCommand(MixerSave)
	Mixer.AddCommand(MixerLoad)
}

func assertNoError(ctx context.Context, err error) {
	if err != nil {
		logger.Panic(ctx, err)
	}
}

func newTinymix(
	ctx context.Context,
	cmd *cobra.Command,
) *tinymix.TinyMix {
	tinymixPath, err := cmd.Flags().GetString("tinymix-path")
	assertNoError(ctx, err)

	audioDeviceIdx, err := cmd.Flags().GetInt("device-idx")
	assertNoError(ctx, err)

	audioDeviceName, err := cmd.Flags().GetString("device-name")
	assertNoError(ctx, err)

	var opts []tinymix.Option

	if tinymixPath != "" {
		opts = append(opts, tinymix.OptionTinymixPath(tinymixPath))
	}
	if audioDeviceIdx >= 0 {
		opts = append(opts, tinymix.OptionDeviceIdx(audioDeviceIdx))
	}
	if audioDeviceName != "" {
		opts = append(opts, tinymix.OptionDeviceName(audioDeviceName))
	}

	c, err := tinymix.New(ctx, opts...)
	assertNoError(ctx, err)
	return c
}

func mixerSave(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()

	c := newTinymix(ctx, cmd)

	state, err := c.GetState(ctx)
	assertNoError(ctx, err)

	b, err := json.Marshal(state)
	assertNoError(ctx, err)

	err = os.WriteFile(args[0], b, 0640)
	assertNoError(ctx, err)
}

func mixerLoad(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()

	c := newTinymix(ctx, cmd)

	b, err := os.ReadFile(args[0])
	assertNoError(ctx, err)

	var state []tinymix.ControlValue
	err = json.Unmarshal(b, &state)
	assertNoError(ctx, err)

	err = c.LoadState(ctx, state)
	assertNoError(ctx, err)
}
