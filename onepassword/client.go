package onepassword

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"strconv"
)

// Struct for Args and path ...
type Client struct {
	Args []string
	path string
}

// Options for client
type ClientOptions struct {
	Account *string
	Cache   *bool
	Config  *string
	Session *string
}

// NewClient creates a new onepassword client.
// opts are the options for the client that are appended to the default Args for the client
// returns a pointer to the client and an error
func NewClient(opts *ClientOptions) (*Client, error) {
	// name of command, flags we always pass
	args := []string{
		"op",
		"--format", "json",
		"--iso-timestamps",
		"--no-color",
	}

	// If passed an account, we set the flag
	if opts.Account != nil {
		args = append(args, "--account", *opts.Account)
	}

	// If using cache, set the flag
	if opts.Cache != nil {
		args = append(args, "--cache", strconv.FormatBool(*opts.Cache))
	}

	// and if we have config, set that flag
	if opts.Config != nil {
		args = append(args, "--config", *opts.Config)
	}

	// if session, set that option too
	if opts.Session != nil {
		args = append(args, "--session", *opts.Session)
	}

	// Figure out the path to the op command (first argument)
	path, err := exec.LookPath(args[0])
	if err != nil {
		return nil, err
	}

	// Return the pointer to our client
	// this will include the Args and path
	return &Client{
		Args: args,
		path: path,
	}, nil
}

// GetJSON gets the JSON from the command and unmarshals it into the data struct provided
// Data is any struct that should match the JSON output
// Args are the arguments to pass to the command
func (c *Client) GetJSON(data any, args ...string) error {
	output, err := c.RunPlain(args...)
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(output), data)
}

func (c *Client) RunPlain(args ...string) (result string, err error) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd := exec.Cmd{
		Path:         c.path,
		Args:         append(c.Args, args...),
		Env:          nil,
		Dir:          "",
		Stdin:        nil,
		Stdout:       stdout,
		Stderr:       nil,
		ExtraFiles:   nil,
		SysProcAttr:  nil,
		Process:      nil,
		ProcessState: nil,
		Err:          nil,
		Cancel:       nil,
		WaitDelay:    0,
	}

	//cmd := exec.Command("sh", "-c", `op item list --vault Adobe --tags AWS --format json`)
	err = cmd.Run()
	if err != nil {
		return stderr.String(), err
	}

	return stdout.String(), nil
}
