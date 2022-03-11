package nsexecutor

import (
	"context"
	"os"
	"strings"

	"github.com/hwameistor/drbd-installer/pkg/exechelper"
	"github.com/hwameistor/drbd-installer/pkg/exechelper/basicexecutor"
)

type nsenterExecutor struct {
	pExecutor   exechelper.Executor
	nsenterArgs []string
}

const nsenterCommand = "nsenter"

// use pid 1 as a target, and enter their mount and pid namespace along with borrowing their root directory
// TODO: Make these configable so that we aren't *always* running everything at max permissions
var nsenterArgsDefalut = []string{"--mount=/proc/1/ns/mnt", "--ipc=/proc/1/ns/ipc", "--net=/proc/1/ns/net", "--"}

// New creates a new nsenterExecutor instance, which implements
// exechelper.Executor interface by wrapping over top of a basic
// executor
func New() exechelper.Executor {
	nsenter := &nsenterExecutor{
		pExecutor: basicexecutor.New(),
	}
	nsenter.setArgsFromEnvOrDefault()
	return nsenter
}

// RunCommand runs a command to completion, and get returns
// If env variable CMD_NSENTER_RUN_ARGS is set, value will set to esenter arg list by default.
func (e *nsenterExecutor) RunCommand(params exechelper.ExecParams) exechelper.ExecResult {
	command := append([]string{params.CmdName}, params.CmdArgs...)
	combinedArgs := append(e.nsenterArgs, command...)
	params.CmdName = nsenterCommand
	params.CmdArgs = combinedArgs
	return e.pExecutor.RunCommand(params)
}

func (e *nsenterExecutor) RunDaemonCommand(ctx context.Context, params exechelper.ExecParams) *exechelper.ExecDaemonResult {
	command := append([]string{params.CmdName}, params.CmdArgs...)
	combinedArgs := append(e.nsenterArgs, command...)
	params.CmdName = nsenterCommand
	params.CmdArgs = combinedArgs
	return e.pExecutor.RunDaemonCommand(ctx, params)
}

// NsenterSetArgs override args of nsenter command
func (e *nsenterExecutor) NsenterSetArgs(args []string) {
	e.nsenterArgs = args
}

// setArgsFromEnvOrDefault override args of nsenter command use env setting or default.
func (e *nsenterExecutor) setArgsFromEnvOrDefault() {
	// args
	runArgsRawStr := os.Getenv("CMD_NSENTER_RUN_ARGS")
	// args separator
	separator := os.Getenv("CMD_NSENTER_ARGS_SEP")
	if len(runArgsRawStr) == 0 {
		e.nsenterArgs = nsenterArgsDefalut
		return
	}

	if len(separator) > 0 {
		e.NsenterSetArgs(strings.Split(runArgsRawStr, separator))
	} else {
		e.NsenterSetArgs([]string{runArgsRawStr})
	}
}
