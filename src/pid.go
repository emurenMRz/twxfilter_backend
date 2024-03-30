package main

import (
	"fmt"
	"os"
	"strconv"
	"syscall"
)

type RunData struct {
	pid         int
	pidFilePath string
}

func (rd *RunData) Close() (err error) {
	err = os.Remove(rd.pidFilePath)
	return
}

func RunDaemon(pidFilename string) (runData RunData, err error) {
	pidFile, err := ExecPath(pidFilename)
	if err != nil {
		return
	}

	isExist, _ := IsExistRunningProcess(pidFile)
	if isExist {
		err = fmt.Errorf("exists running process")
		return
	}

	pid := os.Getpid()
	os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644)

	runData.pid = pid
	runData.pidFilePath = pidFile
	return
}

func IsExistRunningProcess(pidFile string) (bool, error) {
	image, err := os.ReadFile(pidFile)
	if err != nil {
		return false, err
	}

	pid, err := strconv.Atoi(string(image))
	if err != nil {
		return false, err
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return false, err
	}

	err = proc.Signal(syscall.Signal(0))
	if err != nil {
		return false, err
	}

	return true, nil
}
