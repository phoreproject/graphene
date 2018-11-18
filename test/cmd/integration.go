package main

import (
	"os/exec"
	"time"

	"github.com/inconshreveable/log15"
	test "github.com/phoreproject/synapse/test"
)

type Command struct {
	Command string
	Args    []string
}

func main() {
	tests := []struct {
		Test     func() error
		Commands []Command
	}{
		{
			Test: test.TestP2PModuleCommunication,
			Commands: []Command{
				{
					Command: "./synapsep2p",
					Args:    []string{},
				},
				{
					Command: "./synapsep2p",
					Args:    []string{"-rpclisten", "127.0.0.1:11883", "-listen", "/ip4/0.0.0.0/tcp/11881"},
				},
			},
		},
	}

	for _, t := range tests {
		cmds := make([]*exec.Cmd, len(t.Commands))
		for i, c := range t.Commands {
			log15.Info("running command", "cmd", c.Command, "args", c.Args)

			cmd := exec.Command(c.Command, c.Args...)
			b := make([]byte, 1)
			w, err := cmd.StdoutPipe()
			if err != nil {
				panic(err)
			}

			log15.Debug("starting command")

			err = cmd.Start()
			if err != nil {
				panic(err)
			}

			log15.Debug("waiting for first byte printed to stdout")

			_, err = w.Read(b)
			if err != nil {
				panic(err)
			}

			cmds[i] = cmd
		}

		log15.Debug("waiting 2 seconds")

		timer := time.NewTimer(2 * time.Second)
		<-timer.C

		log15.Info("running test")

		err := t.Test()
		if err != nil {
			panic(err)
		}

		log15.Info("tests succeded")

		for _, c := range cmds {
			err := c.Process.Kill()
			if err != nil {
				panic(err)
			}
		}
	}
}
