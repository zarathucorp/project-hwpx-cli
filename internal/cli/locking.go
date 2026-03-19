package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

const documentLockFileName = ".hwpxctl.lock"

type documentLockMetadata struct {
	Command   string `json:"command"`
	PID       int    `json:"pid"`
	Timestamp string `json:"timestamp"`
}

func withMutationLock(targetDir, commandName string, run func() error) error {
	release, err := acquireDocumentLock(targetDir, commandName)
	if err != nil {
		return err
	}
	defer release()
	return run()
}

func acquireDocumentLock(targetDir, commandName string) (func(), error) {
	lockPath := filepath.Join(targetDir, documentLockFileName)

	for attempt := 0; attempt < 2; attempt++ {
		release, err := tryCreateDocumentLock(lockPath, commandName)
		if err == nil {
			return release, nil
		}
		if !os.IsExist(err) {
			return nil, err
		}

		stale, infoText, staleErr := detectStaleDocumentLock(lockPath)
		if staleErr != nil {
			return nil, staleErr
		}
		if stale {
			if err := os.Remove(lockPath); err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			continue
		}

		message := fmt.Sprintf("document is locked by another hwpxctl mutation: %s", infoText)
		return nil, commandError{
			message: message,
			code:    1,
			kind:    "resource_busy",
		}
	}

	return nil, commandError{
		message: "document is locked by another hwpxctl mutation",
		code:    1,
		kind:    "resource_busy",
	}
}

func tryCreateDocumentLock(lockPath, commandName string) (func(), error) {
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}

	metadata := documentLockMetadata{
		Command:   commandName,
		PID:       os.Getpid(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	if err := json.NewEncoder(lockFile).Encode(metadata); err != nil {
		_ = lockFile.Close()
		_ = os.Remove(lockPath)
		return nil, err
	}
	if err := lockFile.Close(); err != nil {
		_ = os.Remove(lockPath)
		return nil, err
	}

	return func() {
		_ = os.Remove(lockPath)
	}, nil
}

func detectStaleDocumentLock(lockPath string) (bool, string, error) {
	content, err := os.ReadFile(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return true, "missing lock file", nil
		}
		return false, "", err
	}

	trimmed := string(content)
	if len(trimmed) == 0 {
		return true, "empty lock metadata", nil
	}

	var metadata documentLockMetadata
	if err := json.Unmarshal(content, &metadata); err != nil {
		return false, fmt.Sprintf("unreadable lock metadata at %s", lockPath), nil
	}

	if metadata.PID <= 0 {
		return true, "invalid lock pid", nil
	}
	if !processExists(metadata.PID) {
		return true, fmt.Sprintf("stale lock from pid=%d", metadata.PID), nil
	}

	infoText := fmt.Sprintf("pid=%d command=%s started=%s", metadata.PID, nonEmptyOr(metadata.Command, "unknown"), nonEmptyOr(metadata.Timestamp, "unknown"))
	return false, infoText, nil
}

func processExists(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}

func nonEmptyOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func lockTargetFromArgs(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

func lockMetadataForTest(commandName string, pid int) []byte {
	metadata := documentLockMetadata{
		Command:   commandName,
		PID:       pid,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	return []byte(fmt.Sprintf("{\"command\":%s,\"pid\":%s,\"timestamp\":%s}\n", strconv.Quote(metadata.Command), strconv.Itoa(metadata.PID), strconv.Quote(metadata.Timestamp)))
}
