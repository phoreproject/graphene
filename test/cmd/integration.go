package main

import (
	"os/exec"
	"time"

	"github.com/sirupsen/logrus"
)

// Command is an integration test command.
type Command struct {
	Command string
	Args    []string
}

func main() {
	tests := []struct {
		Test     func() error
		Commands []Command
	}{}

	for _, t := range tests {
		cmds := make([]*exec.Cmd, len(t.Commands))
		for i, c := range t.Commands {
			logrus.WithField("cmd", c.Command).WithField("args", c.Args).Info("running command")

			cmd := exec.Command(c.Command, c.Args...)
			b := make([]byte, 1)
			w, err := cmd.StderrPipe()
			if err != nil {
				panic(err)
			}

			logrus.Debug("starting command")

			err = cmd.Start()
			if err != nil {
				panic(err)
			}
			logrus.Debug("waiting for first byte printed to stderr")

			_, err = w.Read(b)
			if err != nil {
				panic(err)
			}

			cmds[i] = cmd
		}

		logrus.Debug("waiting 2 seconds")

		timer := time.NewTimer(2 * time.Second)
		<-timer.C

		logrus.Info("running test")

		err := t.Test()
		if err != nil {
			panic(err)
		}

		logrus.Info("tests succeeded")

		for _, c := range cmds {
			err := c.Process.Kill()
			if err != nil {
				panic(err)
			}
		}
	}
}
