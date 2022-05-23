package sleep

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/coder/coder/provisionersdk"
	"github.com/coder/coder/provisionersdk/proto"
)

// Serve starts the echo provisioner.
func Serve(ctx context.Context, options *provisionersdk.ServeOptions) error {
	return provisionersdk.Serve(ctx, &sleep{}, options)
}

const FileName = "sleep"

// The sleep provisioner serves as a dummy provisioner primarily
// used for testing. It reads a duration from the specified file and sleeps for
// that duration, logging every second.
type sleep struct{}

// Parse reads requests from the provided directory to stream responses.
func (*sleep) Parse(_ *proto.Parse_Request, stream proto.DRPCProvisioner_ParseStream) error {
	return stream.Send(&proto.Parse_Response{
		Type: &proto.Parse_Response_Complete{
			Complete: &proto.Parse_Complete{},
		},
	})
}

// Provision reads requests from the provided directory to stream responses.
func (*sleep) Provision(stream proto.DRPCProvisioner_ProvisionStream) error {
	ctx, shutdown := context.WithCancel(stream.Context())
	defer shutdown()

	msg, err := stream.Recv()
	if err != nil {
		return err
	}

	go func() {
		for {
			request, err := stream.Recv()
			if err != nil {
				return
			}

			if request.GetCancel() == nil {
				continue
			}

			shutdown()
			return
		}
	}()

	request := msg.GetStart()
	fi, err := os.ReadFile(filepath.Join(request.Directory, FileName))
	if err != nil {
		return err
	}

	dur, err := time.ParseDuration(string(fi))
	if err != nil {
		return err
	}

	start := time.Now()
	done := time.NewTimer(dur)
	defer done.Stop()
	tick := time.NewTicker(time.Second)
	defer tick.Stop()

	for {
		select {
		case <-done.C:
			return nil

		case <-tick.C:
			_ = stream.Send(&proto.Provision_Response{
				Type: &proto.Provision_Response_Log{
					Log: &proto.Log{
						Level:  proto.LogLevel_INFO,
						Output: fmt.Sprintf("Waiting %s (%s elapsed)", dur.String(), time.Since(start).String()),
					},
				},
			})

		case <-ctx.Done():
			return nil
		}
	}
}

func (*sleep) Shutdown(_ context.Context, _ *proto.Empty) (*proto.Empty, error) {
	return &proto.Empty{}, nil
}

// Tar returns a tar archive of the duration for the sleep provider to sleep.
func Tar(dur time.Duration) ([]byte, error) {
	var buffer bytes.Buffer
	writer := tar.NewWriter(&buffer)

	durStr := dur.String()
	err := writer.WriteHeader(&tar.Header{
		Name: FileName,
		Size: int64(len(durStr)),
	})
	if err != nil {
		return nil, err
	}

	_, err = writer.Write([]byte(durStr))
	if err != nil {
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}
