package exechelper

import (
	"bytes"
	"context"
	"io"
)

// Executor is the interface for executing commands.
type Executor interface {
	RunCommand(params ExecParams) ExecResult
	RunDaemonCommand(ctx context.Context, params ExecParams) *ExecDaemonResult
}

// ExecParams parameters to execute a command
type ExecParams struct {
	CmdName string
	CmdArgs []string
	Timeout int
}

// ExecResult result of executing a command
type ExecResult struct {
	OutBuf   *bytes.Buffer
	ErrBuf   *bytes.Buffer
	ExitCode int
	Error    error
}

// ExecDaemonResult result of executing a command
type ExecDaemonResult struct {
	StdOutPipe io.ReadCloser
	StdErrPipe io.ReadCloser
	ErrCh      chan error
}
