package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

const (
	PidFile = "/tmp/myapp-server.pid"
	LogFile = "/tmp/myapp-server.log"
)

func StartDaemon() error {
	if IsRunning() {
		return fmt.Errorf("server is already running")
	}

	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}

	logFile, err := os.OpenFile(LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to create log file: %v", err)
	}
	defer logFile.Close()

	cmd := exec.Command(executable, "server", "run-daemon")
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start server daemon: %v", err)
	}

	if err := WritePidFile(cmd.Process.Pid); err != nil {
		cmd.Process.Kill()
	}

	return nil
}

func StopDaemon() error {
	if !IsRunning() {
		return fmt.Errorf("server is not running")
	}

	pid, err := ReadPidFile()
	if err != nil {
		return fmt.Errorf("failed to read PID file: %v", err)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %v", err)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM: %", err)
	}

	os.Remove(PidFile)
	return nil
}

func IsRunning() bool {
	pid, err := ReadPidFile()
	if err != nil {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func WritePidFile(pid int) error {
	pidDir := filepath.Dir(PidFile)
	if err := os.Mkdir(pidDir, 0755); err != nil {
		return err
	}

	return os.WriteFile(PidFile, []byte(strconv.Itoa(pid)), 0644)
}

func ReadPidFile() (int, error) {
	data, err := os.ReadFile(PidFile)
	if err != nil {
		return 0, err
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, err
	}

	return pid, nil
}

func GetStatus() string {
	if IsRunning() {
		pid, _ := ReadPidFile()
		return fmt.Sprintf("Server is running (PID: %d", pid)
	}
	return "Server is stopped"
}
