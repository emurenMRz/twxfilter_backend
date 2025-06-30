package main

import (
	"datasource"
	"encoding/json"
	"os"
	"path"
	"path/filepath"
)

func ReadConnectConfig(confName string) (cc datasource.ConnectConfig, err error) {
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

func GetConnection() (conn *datasource.Database, err error) {
	execPath, err := ExecPath("connect.json")
	if err != nil {
		return
	}

	connectConfig, err := ReadConnectConfig(execPath)
	if err != nil {
		return
	}

	conn, err = datasource.Connect(connectConfig)
	if err != nil {
		return
	}

	return
}

func ExecPath(filename string) (execPath string, err error) {
	exe, err := os.Executable()
	if err != nil {
		return
	}

	execPath = path.Join(filepath.Dir(exe), filename)
	return
}
