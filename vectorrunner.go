package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
)

type VectorRunner struct {
	config             string
	outputBuffer       *bytes.Buffer
	cmd                *exec.Cmd
	tempConfigFileName string
}

func NewVectorRunner(config string) *VectorRunner {
	return &VectorRunner{
		config:             config,
		cmd:                nil,
		outputBuffer:       nil,
		tempConfigFileName: "",
	}
}

func (vr *VectorRunner) Start() error {
	tempConfigFile, err := os.CreateTemp("", "VectorDynamicConfig")
	if err != nil {
		return err
	}
	_, err = tempConfigFile.WriteString(vr.config)
	if err != nil {
		return err
	}

	vr.tempConfigFileName = tempConfigFile.Name()

	log.Printf("Starting Vector with arg '-c %s'\n", tempConfigFile.Name())
	cmd := exec.Command("./vector/target/release/vector", "-c", tempConfigFile.Name())

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	vr.outputBuffer = &output

	err = cmd.Start()
	if err != nil {
		return err
	}

	vr.cmd = cmd

	go func() {
		err := cmd.Wait()
		if err != nil {
			log.Panicln("Vector exited with non-nil error:", err)
		} else {
			log.Println("Vector exited normally.")
		}
	}()

	log.Println("Vector has started. Running in pid", cmd.Process.Pid)

	return nil
}

func (vr *VectorRunner) PrintOutputToStdout(prefix string) {
	for {
		if vr.outputBuffer.Len() > 0 {
			line, err := vr.outputBuffer.ReadString('\n')
			if err != nil && err != io.EOF {
				log.Println("vector err:", err)
			} else {
				fmt.Println(prefix, line)
			}
		}
		// Indicates vector has exited
		if vr.cmd.ProcessState != nil {
			return
		}
	}
}

func (vr *VectorRunner) Stop() {
	log.Println("Removing vector config")
	os.Remove(vr.tempConfigFileName)

	if vr.cmd != nil && vr.cmd.ProcessState == nil {
		log.Println("Vector still running, killing now..")
		vr.cmd.Process.Kill()
	}
}
