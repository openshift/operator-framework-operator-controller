package utils

import (
	"bytes"
	"os/exec"
)

func Run(args ...string) (string, error) {
	cmd := exec.Command("oc", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}
