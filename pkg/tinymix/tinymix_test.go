package tinymix

import (
	"context"
	_ "embed"
	"fmt"
	"testing"

	"github.com/facebookincubator/go-belt"
	"github.com/facebookincubator/go-belt/tool/logger"
	"github.com/facebookincubator/go-belt/tool/logger/implementation/zap"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/upstream-dump-sample-0.txt
var tinymixDumpSampleUpstream string

//go:embed testdata/upstream-no-contents-verb.txt
var tinymixDumpErrorUpstreamNoContents string

//go:embed testdata/google-dump-sample-0.txt
var tinymixDumpSampleGoogle string

func TestTinymix(t *testing.T) {
	l := zap.Default()
	ctx := context.Background()
	ctx = logger.CtxWithLogger(ctx, l)
	logger.Default = func() logger.Logger {
		return l
	}
	defer belt.Flush(ctx)

	t.Run("GetState", func(t *testing.T) {
		t.Run("Google", func(t *testing.T) {
			var opts []Option

			opts = append(opts, OptionExec(func(
				ctx context.Context,
				path string,
				args []string,
				stdin []byte,
			) (stdout []byte, stderr []byte, err error) {
				isUpstreamVariant := false
				for _, arg := range args {
					if arg == "contents" {
						isUpstreamVariant = true
						break
					}
				}
				if isUpstreamVariant {
					return nil, []byte("Invalid mixer control: contents"), fmt.Errorf("exit code 2")
				}
				return []byte(tinymixDumpSampleGoogle), []byte{}, nil
			}))

			c, err := New(ctx, opts...)
			if err != nil {
				t.Fatal(err)
			}

			state, err := c.GetState(ctx)
			if err != nil {
				t.Fatal(err)
			}

			require.Len(t, state, 572)
		})

		t.Run("upstream", func(t *testing.T) {
			var opts []Option

			opts = append(opts, OptionExec(func(
				ctx context.Context,
				path string,
				args []string,
				stdin []byte,
			) (stdout []byte, stderr []byte, err error) {
				isUpstreamVariant := false
				for _, arg := range args {
					if arg == "contents" {
						isUpstreamVariant = true
						break
					}
				}
				if !isUpstreamVariant {
					return nil, []byte(tinymixDumpErrorUpstreamNoContents), nil
				}
				return []byte(tinymixDumpSampleUpstream), []byte{}, nil
			}))

			c, err := New(ctx, opts...)
			if err != nil {
				t.Fatal(err)
			}

			state, err := c.GetState(ctx)
			if err != nil {
				t.Fatal(err)
			}

			require.Len(t, state, 572)
		})
	})
}
