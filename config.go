package main

import (
	"bytes"
	"os"
	"text/template"
)

const (
	vectorInputSocketPath  = "/tmp/go-vrl-vectorinput.socket"
	vectorOutputSocketPath = "/tmp/go-vrl-vectoroutput.socket"
)

type VectorConfigTemplateParameters struct {
	VectorInputSocket  string
	VectorOutputSocket string
	VRLProgram         string
}

func GenerateVectorConfig() string {

	data, err := os.ReadFile("./vectorconfig.tmpl")
	if err != nil {
		panic(err)
	}
	tmpl, err := template.New("vector-config").Parse(string(data))
	if err != nil {
		panic(err)
	}

	params := VectorConfigTemplateParameters{
		VectorInputSocket:  vectorInputSocketPath,
		VectorOutputSocket: vectorOutputSocketPath,
		VRLProgram:         `. = replace(string!(.message), r'\b\w{4}\b', "rust")`,
	}

	var out bytes.Buffer
	tmpl.Execute(&out, params)

	return out.String()
}
