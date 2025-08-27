package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"text/template"
)

type ChainParams struct {
	InterfaceName string
	ChainName     string
	FuncName      string
}

var chainTemplate *template.Template

func genFile(file string, data interface{}) {
	var buf bytes.Buffer
	chainTemplate.Execute(&buf, data)
	err := ioutil.WriteFile(file, []byte(buf.String()), 0664)
	if err != nil {
		panic(err)
	}
}

func main() {
	inputFile := os.Args[1]
	chainTemplate = template.Must(template.ParseFiles(inputFile))
	genFile("core/querychain.go", &ChainParams{"DNSQueryHandler", "QueryChain", "HandleQuery"})
	genFile("core/updatechain.go", &ChainParams{"DNSUpdateHandler", "UpdateChain", "HandleUpdate"})
}
