package shared

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

func PrintToPDF(inputPath, outputPath, workspaceDir string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("print-pdf is only supported on macOS")
	}
	if filepath.Ext(strings.ToLower(outputPath)) != ".pdf" {
		return fmt.Errorf("output path must end with .pdf")
	}
	if _, err := os.Stat("/Applications/Hancom Office HWP Viewer.app"); err != nil {
		return fmt.Errorf("Hancom Office HWP Viewer.app is required: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}

	stageDir := workspaceDir
	if stageDir == "" {
		stageDir = filepath.Dir(outputPath)
	}
	if err := os.MkdirAll(stageDir, 0o755); err != nil {
		return err
	}

	stageBase := fmt.Sprintf("hwpxctl-print-%d", time.Now().UnixNano())
	stageFile := filepath.Join(stageDir, stageBase+".pdf")
	_ = os.Remove(stageFile)

	workspaceName := filepath.Base(stageDir)
	sourceDir := filepath.Base(filepath.Dir(inputPath))
	docName := filepath.Base(inputPath)

	if err := runHancomPrintScript(inputPath, docName, workspaceName, sourceDir, stageBase); err != nil {
		return err
	}

	foundPath, err := waitForPrintedPDF(stageBase+".pdf", stageDir, filepath.Dir(inputPath))
	if err != nil {
		return err
	}
	defer os.Remove(foundPath)

	if err := os.Rename(foundPath, outputPath); err == nil {
		return nil
	}
	if err := copyFile(foundPath, outputPath); err != nil {
		return err
	}
	return os.Remove(foundPath)
}

func runHancomPrintScript(inputPath, docName, workspaceName, sourceDir, stageBase string) error {
	script := `
on clickMenuItemIfExists(theButton, itemName)
	tell application "System Events"
		tell process "Hancom Office HWP Viewer"
			try
				click theButton
				delay 0.4
				click menu item itemName of menu 1 of theButton
				return true
			on error
				return false
			end try
		end tell
	end tell
end clickMenuItemIfExists

on describeOpenWindows()
	tell application "System Events"
		tell process "Hancom Office HWP Viewer"
			set windowDescriptions to {}
			repeat with w in windows
				set windowName to ""
				set windowText to ""
				try
					set windowName to name of w
				end try
				try
					set textValues to value of static texts of w
					if (count of textValues) > 0 then
						set windowText to item 1 of textValues
					end if
				end try
				if windowText is not "" then
					set end of windowDescriptions to windowName & ": " & windowText
				else
					set end of windowDescriptions to windowName
				end if
			end repeat
			return windowDescriptions as string
		end tell
	end tell
end describeOpenWindows

on run argv
	set inputPath to item 1 of argv
	set docName to item 2 of argv
	set workspaceName to item 3 of argv
	set sourceDirName to item 4 of argv
	set stageBase to item 5 of argv

	try
		tell application "Hancom Office HWP Viewer" to quit
	end try
	delay 2
	do shell script "open -a " & quoted form of "/Applications/Hancom Office HWP Viewer.app" & " " & quoted form of inputPath
	delay 4

	tell application "System Events"
		tell process "Hancom Office HWP Viewer"
			set frontmost to true
			set targetWindow to missing value
			repeat 40 times
				repeat with w in windows
					if name of w is docName then
						set targetWindow to w
						exit repeat
					end if
				end repeat
				if targetWindow is not missing value then exit repeat
				delay 0.5
			end repeat
			if targetWindow is missing value then error "viewer window not found: " & my describeOpenWindows()

			click menu item "인쇄..." of menu "파일" of menu bar item "파일" of menu bar 1
			repeat 40 times
				if exists sheet 1 of targetWindow then exit repeat
				delay 0.5
			end repeat
			if not (exists sheet 1 of targetWindow) then error "print dialog did not open"

			set printSheet to sheet 1 of targetWindow
			set pdfButton to menu button 1 of group 2 of splitter group 1 of printSheet
			click pdfButton
			delay 0.4
			click menu item "PDF로 저장…" of menu 1 of pdfButton

			repeat 40 times
				if exists sheet 1 of printSheet then exit repeat
				delay 0.5
			end repeat
			if not (exists sheet 1 of printSheet) then error "pdf save dialog did not open"

			set saveSheet to sheet 1 of printSheet
			set saveGroup to splitter group 1 of saveSheet

			set locationButton to pop up button "위치:" of saveGroup
			set selectedLocation to false
			if workspaceName is not "" then
				set selectedLocation to my clickMenuItemIfExists(locationButton, workspaceName)
			end if
			if (not selectedLocation) and sourceDirName is not "" then
				set selectedLocation to my clickMenuItemIfExists(locationButton, sourceDirName)
			end if
			delay 0.8

			click text field "별도 저장:" of saveGroup
			delay 0.2
			keystroke "a" using {command down}
			delay 0.2
			keystroke stageBase
			delay 0.4
			click button "저장" of saveGroup

			repeat 20 times
				if exists window "" then
					try
						click button "확인" of window ""
					end try
					error "print dialog reported an error"
				end if
				if not (exists saveSheet) then exit repeat
				delay 0.5
			end repeat
		end tell
	end tell
end run
`

	cmd := exec.Command("osascript", "-", inputPath, docName, workspaceName, sourceDir, stageBase)
	cmd.Stdin = strings.NewReader(script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			return err
		}
		return fmt.Errorf("%w: %s", err, trimmed)
	}
	return nil
}

func waitForPrintedPDF(fileName string, candidateDirs ...string) (string, error) {
	dirs := uniqueDirs(candidateDirs)
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		for _, dir := range dirs {
			if dir == "" {
				continue
			}
			path := filepath.Join(dir, fileName)
			info, err := os.Stat(path)
			if err == nil && info.Size() > 0 {
				return path, nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return "", fmt.Errorf("printed pdf was not created")
}

func uniqueDirs(values []string) []string {
	seen := map[string]struct{}{}
	dirs := make([]string, 0, len(values)+3)
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		dirs = append(dirs, value)
	}

	home, err := os.UserHomeDir()
	if err == nil {
		for _, extra := range []string{
			filepath.Join(home, "Documents"),
			filepath.Join(home, "Desktop"),
			filepath.Join(home, "Downloads"),
		} {
			if _, ok := seen[extra]; ok {
				continue
			}
			seen[extra] = struct{}{}
			dirs = append(dirs, extra)
		}
	}

	sort.Strings(dirs)
	return dirs
}
