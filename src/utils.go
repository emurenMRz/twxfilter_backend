package main

import (
	"encoding/json"
	"os"
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
	connectConfig, err := ReadConnectConfig("connect.json")
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
