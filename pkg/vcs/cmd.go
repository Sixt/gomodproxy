package vcs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type cmdVCS struct {
	log           logger
	module        string
	moduleEncoded string
	cmd           string
}

func encodeBangs(s string) string {
	buf := []byte{}
	for _, r := range s {
		if 'A' <= r && r <= 'Z' {
			buf = append(buf, '!', byte(r+'a'-'A'))
		} else {
			buf = append(buf, byte(r))
		}
	}
	return string(buf)
}

func NewCommand(l logger, cmd string, module string) VCS {
	return &cmdVCS{log: l, cmd: cmd, module: module, moduleEncoded: encodeBangs(module)}
}

func (c *cmdVCS) List(ctx context.Context) ([]Version, error) {
	b, err := c.exec(ctx,
		"MODULE="+c.module,
		"MODULE_ENCODED="+c.moduleEncoded,
		"ACTION=list",
		"VERSION=latest",
		"FILEPATH="+c.module+"/@v/list",
		"FILEPATH_ENCODED="+c.moduleEncoded+"/@v/list",
	)
	if err != nil {
		return nil, err
	}
	versions := []Version{}
	for _, line := range strings.Split(string(b), "\n") {
		versions = append(versions, Version(line))
	}
	return versions, nil
}

func (c *cmdVCS) Timestamp(ctx context.Context, version Version) (time.Time, error) {
	b, err := c.exec(ctx,
		"MODULE="+c.module,
		"MODULE_ENCODED="+c.moduleEncoded,
		"ACTION=timestamp",
		"VERSION="+version.String(),
		"FILEPATH="+c.module+"/@v/"+version.String()+".info",
		"FILEPATH_ENCODED="+c.moduleEncoded+"/@v/"+version.String()+".info",
	)
	if err != nil {
		return time.Time{}, err
	}
	info := struct {
		Version string
		Time    time.Time
	}{}
	if json.Unmarshal(b, &info) == nil {
		return info.Time, nil
	}
	if t, err := time.Parse(time.RFC3339, string(b)); err == nil {
		return t, nil
	}
	if sec, err := strconv.ParseInt(string(b), 10, 64); err == nil {
		return time.Unix(sec, 0), nil
	}
	return time.Time{}, errors.New("unknown time format")
}

func (c *cmdVCS) Zip(ctx context.Context, version Version) (io.ReadCloser, error) {
	b, err := c.exec(ctx,
		"MODULE="+c.module,
		"MODULE_ENCODED="+c.moduleEncoded,
		"ACTION=zip",
		"VERSION="+version.String(),
		"FILEPATH="+c.module+"/@v/"+version.String()+".zip",
		"FILEPATH_ENCODED="+c.moduleEncoded+"/@v/"+version.String()+".zip",
	)
	if err != nil {
		return nil, err
	}
	return ioutil.NopCloser(bytes.NewReader(b)), nil
}

func (c *cmdVCS) exec(ctx context.Context, env ...string) ([]byte, error) {
	cmd := exec.Command("sh", "-c", c.cmd)
	cmd.Env = append(os.Environ(), env...)
	cmd.Stderr = os.Stderr
	return cmd.Output()
}
