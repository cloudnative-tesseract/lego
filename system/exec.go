package system

import (
	"bytes"
	"context"
	"os/exec"
	"os/user"
	"strings"
	"time"

	"github.com/pkg/errors"
	"k8s.io/klog/v2"
)

const (
	info  = 0b1
	debug = 0b10
)

type ExecuteResult struct {
	ExitCode int
	Output   string
}

func (r ExecuteResult) IsSuccessful() bool {
	return r.ExitCode == 0
}

func (r ExecuteResult) AsError() error {
	if r.IsSuccessful() {
		return nil
	}
	return errors.Errorf("failed to execute command, exitCode: %d, output: %s", r.ExitCode, r.Output)
}

func (r ExecuteResult) Lines() []string {
	if len(r.Output) == 0 {
		return []string{}
	}
	if !strings.Contains(r.Output, "\n") {
		return []string{r.Output}
	}
	lines := strings.Split(strings.Trim(r.Output, "\n"), "\n")
	return lines
}

// Execute the given command and expect the command to succeed (exits with 0)
// If the command exits with a non-zero code, return an error
func (c *command) Execute() (*ExecuteResult, error) {
	executeResult, err := c.execute(info)
	if err != nil {
		return executeResult, err
	}
	return executeResult, executeResult.AsError()
}

// ExecuteAllowFailure the given command, allow the command to failed (exits with non-zero code)
func (c *command) ExecuteAllowFailure() (*ExecuteResult, error) {
	return c.execute(info)
}

func (c *command) ExecuteWithDebug() (*ExecuteResult, error) {
	executeResult, err := c.execute(debug)
	if err != nil {
		return executeResult, err
	}
	return executeResult, executeResult.AsError()
}

// Execute the given command with the given program and timeout
// It returns:
// 1. the exit code
// 2. the combined output of stdout and stderr
// 3. the error
func (c *command) execute(flag int) (*ExecuteResult, error) {
	if c.context == nil {
		c.context = context.Background()
	}
	if flag&debug != 0 {
		klog.Infof("execute shell command start, command=%s", c.String())
	} else {
		klog.Infof("execute shell command start, command=%s", c.String())
	}
	var runCmd *exec.Cmd
	currentUser := getCurrentUser()
	if c.user == "" || c.user == currentUser {
		runCmd = exec.Command(string(c.program), "-c", c.cmd)
	} else if currentUser == RootUser {
		runCmd = exec.Command("runuser", "-l", c.user, "-c", c.cmd)
	} else if c.user == RootUser {
		runCmd = exec.Command("sudo", string(c.program), "-c", c.cmd)
	} else {
		runCmd = exec.Command("sudo", "-u", c.user, string(c.program), "-c", c.cmd)
	}
	b, err := CombinedOutputTimeout(runCmd, c.timeout)
	output := string(b)
	klog.Infof("execute shell command %s, output=%s", c.String(), output)
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode := exitError.ExitCode()
			klog.Infof("execute shell command failed, command=%s, exitCode=%d", c.String(), exitCode)
			return &ExecuteResult{
				ExitCode: exitCode,
				Output:   output,
			}, nil
		} else {
			klog.Errorf("execute shell command error, command=%s, error=%s", c.String(), err)
			return nil, errors.Errorf("error when execute shell command %s: %s", c.cmd, err)
		}
	} else {
		if flag&debug != 0 {
			klog.Infof("execute shell command end, command=%s", c.String())
		} else {
			klog.Infof("execute shell command end, command=%s", c.String())
		}
		return &ExecuteResult{
			ExitCode: 0,
			Output:   output,
		}, nil
	}
}

// CombinedOutputTimeout runs the given command with the given timeout and returns the output of stdout and stderr
// If the command times out, it attempts to kill the process
func CombinedOutputTimeout(c *exec.Cmd, timeout time.Duration) ([]byte, error) {
	var b bytes.Buffer
	c.Stdout = &b
	c.Stderr = &b
	if err := c.Start(); err != nil {
		return nil, err
	}
	err := WaitTimeout(c, timeout)
	return b.Bytes(), err
}

// StdoutOutputTimeout runs the given command with the given timeout and returns the output of stdout
// If the command times out, it attempts to kill the process
func StdoutOutputTimeout(c *exec.Cmd, timeout time.Duration) ([]byte, error) {
	var b bytes.Buffer
	c.Stdout = &b
	c.Stderr = nil
	if err := c.Start(); err != nil {
		return nil, err
	}
	err := WaitTimeout(c, timeout)
	return b.Bytes(), err
}

// RunTimeout runs the given command with the given timeout
// If the command times out, it attempts to kill the process
func RunTimeout(c *exec.Cmd, timeout time.Duration) error {
	if err := c.Start(); err != nil {
		return err
	}
	return WaitTimeout(c, timeout)
}

func getCurrentUser() string {
	currentUser, err := user.Current()
	if err != nil {
		return ""
	}
	return currentUser.Username
}
