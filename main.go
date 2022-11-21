package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"time"
)

const (
	fakeLogLine = "Oct 17 14:33:33 | XSS | ERROR | (/viral/interactive/deliverables/holistic.go:3) | sed et dolorem minima et corrupti abcd veniam qui blanditiis optio explicabo et amet qui sint ut iure neque eveniet quod odio distinctio quas veniam voluptatibus quibusdam esse maiores dolores magni numquam sed deserunt quia odio fuga deserunt cumque a aliquam ad dolores dolore aut sapiente necessitatibus ut autem necessitatibus quam eveniet et omnis aut quos dolorem culpa nostrum quas provident tempora voluptate iure quos iste consequatur minima accusantium molestiae consequatur perspiciatis quis quia at incidunt non veritatis deserunt totam iure autem asperiores rerum officiis iusto et explicabo sunt et rerum molestiae hic dolore neque eum vel rerum perspiciatis autem et consequuntur consequatur aliquam dolore magni ea est illum accusamus rerum magnam neque odio voluptatibus est temporibus quo ullam nobis soluta quo ipsum temporibus perferendis et esse repellendus ea id explicabo nostrum repellat vero perferendis possimus optio consectetur deserunt aspern"
	Purple      = "\033[35m"
	Reset       = "\033[0m"
)

func listenForeverAndPrintThroughput(reader *bufio.Reader) {
	blackhole := BlackholeRecorder[[]byte]{}

	go func() {
		for {
			log.Println(blackhole.AvgThroughput())
			time.Sleep(3 * time.Second)
		}
	}()

	for {
		line, _, err := reader.ReadLine()
		if err == io.EOF {
			// Nothing to read
		} else if err != nil {
			log.Panicln(err)
		} else {
			// Read something useful
			blackhole.Consume(line)
			fmt.Println(string(line))
		}
	}
}

func startVectorWithConfig(config string) (*exec.Cmd, *bytes.Buffer, error) {
	tempConfigFile, err := os.CreateTemp("", "VectorDynamicConfig")
	if err != nil {
		return nil, nil, err
	}
	_, err = tempConfigFile.WriteString(config)
	if err != nil {
		return nil, nil, err
	}
	log.Printf("Starting Vector with arg '-c %s'\n", tempConfigFile.Name())
	cmd := exec.Command("./vector/target/release/vector", "-c", tempConfigFile.Name())
	// defer os.Remove(tempConfigFile.Name())

	var output bytes.Buffer
	cmd.Stdout = &output

	err = cmd.Start()
	if err != nil {
		return nil, nil, err
	}

	return cmd, &output, nil
}

func sendFakeLogDataForever(wr io.Writer) {
	for {
		wr.Write([]byte(fakeLogLine))
	}
}

func main() {
	config := GenerateVectorConfig()

	log.Println("Going to start vector...")
	cmd, stdoutBytes, err := startVectorWithConfig(config)
	if err != nil {
		log.Panicln(err)
	}
	log.Printf("Vector has started succesfully, running in PID %d\n", cmd.Process.Pid)

	go func() {
		//fmt.Println("Vector output will be printed in " + Purple + "Purple" + Reset)
		fmt.Println("Vector output:")
		for {
			if stdoutBytes.Len() > 0 {
				line, err := stdoutBytes.ReadString('\n')
				if err == nil {
					fmt.Println("vector:", err)
				} else {
					fmt.Println("vector: ", line)
				}
			}
		}
	}()

	// By this point vector should have already started or will start shortly, so
	// open a socket and wait for vector to connect
	reader, err := ListenOnUDSSocket(vectorOutputSocketPath)
	if err != nil {
		log.Panicln(err)
	}

	// Once vector connects, listen and print throughput stats
	go listenForeverAndPrintThroughput(reader)

	// Once vector has connected to `vectorOutputSocketPath`,
	// lets try 5 times to connect to `vectorInputSocketPath`
	// Connect to vector input socket and start sending data
	writer, err := ConnectToUDSSocket(vectorInputSocketPath, 5)
	if err != nil {
		log.Panicln(err)
	}

	sendFakeLogDataForever(writer)
}
