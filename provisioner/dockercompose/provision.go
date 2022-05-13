package dockercompose

import (
	"bufio"
	"context"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/coder/coder/provisionersdk"
	"github.com/coder/coder/provisionersdk/proto"
)

func (p *provisionerServer) Provision(stream proto.DRPCProvisioner_ProvisionStream) error {
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	req, err := stream.Recv()
	if err != nil {
		return err
	}
	if req.GetCancel() != nil {
		return nil
	}
	start := req.GetStart()
	if start == nil {
		return nil
	}
	go listenForCancel(stream, cancel)

	env := getEnv(start)
	resources, err := getResources(ctx, start, env)
	if start.GetDryRun() {
		return complete(stream, resources)
	}

	cmd := exec.CommandContext(ctx, "docker", getArgs(start)...)
	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()
	cmd.Stderr = stderrW
	cmd.Stdout = stdoutW
	cmd.Dir = start.GetDirectory()
	cmd.Env = env
	go logFromReader(stream, stderrR)
	go logFromReader(stream, stdoutR)
	if err := cmd.Run(); err != nil {
		return err
	}
	return complete(stream, resources)
}

func getArgs(start *proto.Provision_Start) []string {
	args := []string{"compose"}
	switch start.GetMetadata().GetWorkspaceTransition() {
	case proto.WorkspaceTransition_DESTROY:
		args = append(args, "down", "-v")
	case proto.WorkspaceTransition_START:
		args = append(args, "up", "--remove-orphans", "-d")
	case proto.WorkspaceTransition_STOP:
		args = append(args, "stop")
	}
	return args
}

func getEnv(start *proto.Provision_Start) []string {
	env := os.Environ()
	env = append(env,
		"CODER_AGENT_URL="+start.Metadata.CoderUrl,
		"CODER_WORKSPACE_TRANSITION="+strings.ToLower(start.Metadata.WorkspaceTransition.String()),
		"CODER_WORKSPACE_NAME="+start.Metadata.WorkspaceName,
		"CODER_WORKSPACE_OWNER="+start.Metadata.WorkspaceOwner,
		"CODER_WORKSPACE_ID="+start.Metadata.WorkspaceId,
		"CODER_WORKSPACE_OWNER_ID="+start.Metadata.WorkspaceOwnerId,
	)
	for key, value := range provisionersdk.AgentScriptEnv() {
		env = append(env, key+"="+value)
	}

	for _, param := range start.GetParameterValues() {
		if param.GetDestinationScheme() == proto.ParameterDestination_ENVIRONMENT_VARIABLE {
			env = append(env, param.GetName()+"="+param.GetValue())
		}
	}
	return env
}

func listenForCancel(stream proto.DRPCProvisioner_ProvisionStream, cancel func()) {
	defer cancel()
	for {
		req, err := stream.Recv()
		if err != nil {
			return
		}
		if req.GetCancel() != nil {
			return
		}
		// non-cancel message from provisionerd?
		panic(req)
	}
}

func logFromReader(stream proto.DRPCProvisioner_ProvisionStream, stdout io.Reader) {
	s := bufio.NewScanner(stdout)
	for s.Scan() {
		if s.Err() != nil {
			_ = stream.Send(&proto.Provision_Response{
				Type: &proto.Provision_Response_Log{
					Log: &proto.Log{
						Level:  proto.LogLevel_ERROR,
						Output: "unable to scan docker compose output",
					},
				},
			})
			return
		}
		_ = stream.Send(&proto.Provision_Response{
			Type: &proto.Provision_Response_Log{
				Log: &proto.Log{
					Level:  proto.LogLevel_INFO,
					Output: s.Text(),
				},
			},
		})
	}
}

func complete(stream proto.DRPCProvisioner_ProvisionStream, resources []*proto.Resource) error {
	return stream.Send(&proto.Provision_Response{
		Type: &proto.Provision_Response_Complete{
			Complete: &proto.Provision_Complete{
				Resources: resources,
			},
		},
	})
}

func getResources(ctx context.Context, start *proto.Provision_Start, env []string) ([]*proto.Resource, error) {
	if start.GetMetadata().GetWorkspaceTransition() == proto.WorkspaceTransition_DESTROY {
		// duh, no resources on destroy.
		return nil, nil
	}
	var resources []*proto.Resource

	// volumes
	cmd := exec.CommandContext(ctx, "docker", "compose", "config", "--volumes")
	cmd.Env = env
	cmd.Dir = start.GetDirectory()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	s := bufio.NewScanner(stdout)
	for s.Scan() {
		resources = append(resources, &proto.Resource{
			Name:   s.Text(),
			Type:   "volume",
			Agents: nil,
		})
	}
	if err := cmd.Wait(); err != nil {
		return nil, err
	}

	if start.GetMetadata().GetWorkspaceTransition() == proto.WorkspaceTransition_STOP {
		// only volumes are retained on STOP
		return resources, nil
	}

	// services
	cmd = exec.CommandContext(ctx, "docker", "compose", "config", "--services")
	cmd.Env = env
	cmd.Dir = start.GetDirectory()
	stdout, err = cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	s = bufio.NewScanner(stdout)
	for s.Scan() {
		resources = append(resources, &proto.Resource{
			Name:   s.Text(),
			Type:   "service",
			Agents: nil,
		})
	}
	if err := cmd.Wait(); err != nil {
		return nil, err
	}
	return resources, nil
}
