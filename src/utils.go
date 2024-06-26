package main

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func ReadConnectConfig(confName string) (cc ConnectConfig, err error) {
	in, err := os.ReadFile(confName)
	if err != nil {
		return
	}

	err = json.Unmarshal(in, &cc)
	if err != nil {
		return
	}

	return
}

func GetConnection() (conn *Database, err error) {
	execPath, err := ExecPath("connect.json")
	if err != nil {
		return
	}

	connectConfig, err := ReadConnectConfig(execPath)
	if err != nil {
		return
	}

	conn, err = Connect(connectConfig)
	if err != nil {
		return
	}

	return
}

func GetSelfName() string {
	t := strings.Split(os.Args[0], "/")
	return t[len(t)-1]
}

func ExecPath(filename string) (execPath string, err error) {
	exe, err := os.Executable()
	if err != nil {
		return
	}

	execPath = path.Join(filepath.Dir(exe), filename)
	return
}
