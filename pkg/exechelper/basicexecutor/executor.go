package basicexecutor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/hwameistor/drbd-installer/pkg/exechelper"
	log "github.com/sirupsen/logrus"
)

type basicExecutor struct {
	formatRegex *regexp.Regexp
}

const (
	defaultExecTimeout = 30

	exitCodeTimeout    = 124
	exitCodeErrDefault = 1
	exitCodeSuccess    = 0
)

// New creates a new basicExecutor instance, which implements
// exechelper.Executor interface
func New() exechelper.Executor {
	return &basicExecutor{}
}

func (e *basicExecutor) squashString(str string) string {
	if e.formatRegex == nil {
		e.formatRegex = regexp.MustCompile("[\t\n\r]+")
	}
	return e.formatRegex.ReplaceAllString(str, " ")
}

// RunCommand run a command, and get result
func (e *basicExecutor) RunCommand(params exechelper.ExecParams) exechelper.ExecResult {
	log.WithFields(log.Fields{"params": params}).Debug("Running command")

	// Create a new timeout context
	if params.Timeout == 0 {
		params.Timeout = defaultExecTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(params.Timeout))
	defer cancel()

	outbuf, errbuf := bytes.NewBufferString(""), bytes.NewBufferString("")
	cmd := exec.CommandContext(ctx, params.CmdName, params.CmdArgs...)
	cmd.Stdout = outbuf
	cmd.Stderr = errbuf
	err := cmd.Run()

	result := exechelper.ExecResult{
		OutBuf:   bytes.NewBufferString(strings.TrimSuffix(outbuf.String(), "\n")),
		ErrBuf:   bytes.NewBufferString(strings.TrimSuffix(errbuf.String(), "\n")),
		ExitCode: exitCodeSuccess,
		Error:    err,
	}

	if ctx.Err() == context.DeadlineExceeded {
		result.ExitCode = exitCodeTimeout
		result.Error = fmt.Errorf("Command %s %s timed out after %d seconds", params.CmdName, params.CmdArgs, params.Timeout)
		err = result.Error
	}

	if err != nil {
		// try to get the exit code
		if exitError, ok := err.(*exec.ExitError); ok {
			ws := exitError.Sys().(syscall.WaitStatus)
			result.ExitCode = ws.ExitStatus()
		} else {
			// failed to get exit code, use default code
			result.ExitCode = exitCodeErrDefault
		}
		result.Error = errors.New(e.squashString(err.Error()))
	}

	log.Debug("Finished running command")
	return result
}

// RunDaemonCommand run a command as daemon, and return STDOUT and STDERR pipes
func (e *basicExecutor) RunDaemonCommand(ctx context.Context, params exechelper.ExecParams) *exechelper.ExecDaemonResult {
	result := &exechelper.ExecDaemonResult{
		ErrCh: make(chan error, 1),
	}

	cmd := exec.Command(params.CmdName, params.CmdArgs...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.WithError(err).Errorf("Failed to get STDOUT pipe of %s", params.CmdName)
		result.ErrCh <- err
		return result
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.WithError(err).Errorf("Failed to get STDERR pipe of %s", params.CmdName)
		result.ErrCh <- err
		return result
	}
	result.StdOutPipe = stdout
	result.StdErrPipe = stderr

	if err := cmd.Start(); err != nil {
		log.WithError(err).Errorf("Failed to start daemon %s", params.CmdName)
		result.ErrCh <- err
		return result
	}

	go func() {
		<-ctx.Done()
		if err := cmd.Wait(); err != nil {
			log.WithError(err).Errorf("Failed to wait daemon %s", params.CmdName)
			result.ErrCh <- err
		}
		if cmd.ProcessState != nil && !cmd.ProcessState.Exited() && cmd.Process != nil {
			if err := cmd.Process.Kill(); err != nil {
				log.WithError(err).Errorf("Failed to stop daemon %s", params.CmdName)
				result.ErrCh <- err
			} else {
				if err := stdout.Close(); err != nil {
					log.WithError(err).Errorf("Failed to close STDOUT pipe of %s", params.CmdName)
					result.ErrCh <- err
				}
				if err := stderr.Close(); err != nil {
					log.WithError(err).Errorf("Failed to close STDERR pipe of %s", params.CmdName)
					result.ErrCh <- err
				}
			}
		}
	}()

	return result
}
