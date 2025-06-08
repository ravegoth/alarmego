package main

import (
	"bufio"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func main() {
	const alarmFile = "./alarms.txt"
	args := os.Args[1:]

	if len(args) > 0 && (args[0] == "add" || args[0] == "addo") {
		if len(args) < 3 {
			if len(args) < 2 {
				print("Error: missing interval\n")
			} else {
				print("Error: missing text\n")
			}
			return
		}
		interval := args[1]
		text := strings.Join(args[2:], " ")
		oneTime := args[0] == "addo"
		addAlarm(alarmFile, interval, text, oneTime)
		return
	}

	if len(args) > 0 && args[0] == "remove" {
		if len(args) < 2 {
			print("Error: missing text\n")
			return
		}
		text := strings.Join(args[1:], " ")
		removeClosestAlarm(alarmFile, text)
		return
	}

	ensureAlarmFile(alarmFile)
	alarms := readAlarms(alarmFile)
	if len(alarms) == 0 {
		print("No alarms to schedule.\n")
		return
	}

	for _, alarm := range alarms {
		go scheduleAlarm(alarm.interval, alarm.message, alarm.oneTime)
	}

	select {}
}

type alarm struct {
	interval time.Duration
	message  string
	oneTime  bool
}

func addAlarm(filename, intervalStr, message string, oneTime bool) {
	prefix := ""
	if oneTime {
		prefix = "o"
	}
	line := prefix + intervalStr + `=>"` + message + `"` + "\n"
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		print("Error adding alarm: ", err.Error(), "\n")
		return
	}
	defer f.Close()
	if _, err := f.WriteString(line); err != nil {
		print("Error writing to alarm file: ", err.Error(), "\n")
	} else {
		print("Alarm added successfully.\n")
	}
}

func ensureAlarmFile(filename string) {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) || (err == nil && info.Size() == 0) {
		defaultContent := `10m=>"Reminder to configure alarmego reminders"`
		err := os.WriteFile(filename, []byte(defaultContent), 0644)
		if err != nil {
			print("Error creating default alarm file: ", err.Error(), "\n")
			os.Exit(1)
		}
		print("Default alarm file created.\n")
	} else if err != nil {
		print("Error accessing alarm file: ", err.Error(), "\n")
		os.Exit(1)
	}
}

func readAlarms(filename string) []alarm {
	file, err := os.Open(filename)
	if err != nil {
		print("Error opening alarm file: ", err.Error(), "\n")
		os.Exit(1)
	}
	defer file.Close()

	var alarms []alarm
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		interval, message, err := parseAlarmLine(line)
		if err != nil {
			print("Error parsing line ", strconv.Itoa(lineNumber), ": ", err.Error(), "\n")
			continue
		}
		oneTime := strings.HasPrefix(line, "o")
		alarms = append(alarms, alarm{interval: interval, message: message, oneTime: oneTime})
	}

	if err := scanner.Err(); err != nil {
		print("Error reading alarm file: ", err.Error(), "\n")
		os.Exit(1)
	}

	print("Successfully read ", strconv.Itoa(len(alarms)), " alarms.\n")
	return alarms
}

func parseAlarmLine(line string) (time.Duration, string, error) {
	pattern := `^(o?)(.+)=>"(.*)"$`
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(line)
	if len(matches) != 4 {
		return 0, "", errInvalidFormat()
	}

	intervalStr := strings.TrimSpace(matches[2])
	message := matches[3]

	interval, err := parseDuration(intervalStr)
	if err != nil {
		return 0, "", errInvalidInterval(intervalStr, err)
	}

	print("Found reminder: '", message, "'\n")
	return interval, message, nil
}

func parseDuration(s string) (time.Duration, error) {
	var totalDuration time.Duration
	pattern := `(\d+)([hms])`
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(s, -1)
	if matches == nil {
		return 0, errCouldNotParseDuration()
	}
	for _, match := range matches {
		value, _ := strconv.Atoi(match[1])
		unit := match[2]
		switch unit {
		case "h":
			totalDuration += time.Duration(value) * time.Hour
		case "m":
			totalDuration += time.Duration(value) * time.Minute
		case "s":
			totalDuration += time.Duration(value) * time.Second
		}
	}
	return totalDuration, nil
}

func scheduleAlarm(interval time.Duration, message string, oneTime bool) {
	if oneTime {
		time.Sleep(interval)
		print("One-time reminder: ", message, "\n")
		sendNotification("Alarmego", message)
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			print("Reminding: ", message, "\n")
			sendNotification("Alarmego", message)
		}
	}
}

func sendNotification(title, message string) {
	script := `
        [Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime]
        $template = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02)
        $texts = $template.GetElementsByTagName("text")
        $texts.Item(0).AppendChild($template.CreateTextNode("` + title + `")) | Out-Null
        $texts.Item(1).AppendChild($template.CreateTextNode("` + message + `")) | Out-Null
        $toast = [Windows.UI.Notifications.ToastNotification]::new($template)
        $notifier = [Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("Alarmego")
        $notifier.Show($toast)
    `
	cmd := exec.Command("powershell", "-Command", script)
	err := cmd.Run()
	if err != nil {
		print("Error sending notification: ", err.Error(), "\n")
	}
}

func removeClosestAlarm(filename, target string) {
	file, err := os.Open(filename)
	if err != nil {
		print("Error opening alarm file: ", err.Error(), "\n")
		return
	}
	defer file.Close()

	var lines []string
	var messages []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)

		_, msg, err := parseAlarmLine(line)
		if err == nil {
			messages = append(messages, msg)
		} else {
			messages = append(messages, "")
		}
	}

	if len(messages) == 0 {
		print("No alarms found to remove.\n")
		return
	}

	minDist := -1
	minIndex := -1
	for i, msg := range messages {
		if msg == "" {
			continue
		}
		d := levenshtein(msg, target)
		if minDist == -1 || d < minDist {
			minDist = d
			minIndex = i
		}
	}

	if minIndex == -1 {
		print("No matching alarm found.\n")
		return
	}

	lines = append(lines[:minIndex], lines[minIndex+1:]...)
	err = os.WriteFile(filename, []byte(strings.Join(lines, "\n")+"\n"), 0644)
	if err != nil {
		print("Error updating alarm file: ", err.Error(), "\n")
		return
	}

	print("Removed closest alarm: ", messages[minIndex], "\n")
}

func levenshtein(a, b string) int {
	aLen := len(a)
	bLen := len(b)

	dp := make([][]int, aLen+1)
	for i := range dp {
		dp[i] = make([]int, bLen+1)
	}

	for i := 0; i <= aLen; i++ {
		dp[i][0] = i
	}
	for j := 0; j <= bLen; j++ {
		dp[0][j] = j
	}

	for i := 1; i <= aLen; i++ {
		for j := 1; j <= bLen; j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			dp[i][j] = min(dp[i-1][j]+1, min(dp[i][j-1]+1, dp[i-1][j-1]+cost))
		}
	}

	return dp[aLen][bLen]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func errInvalidFormat() error {
	return &customError{"invalid format"}
}
func errInvalidInterval(intervalStr string, err error) error {
	return &customError{"invalid interval '" + intervalStr + "': " + err.Error()}
}
func errCouldNotParseDuration() error {
	return &customError{"could not parse duration"}
}

type customError struct {
	message string
}

func (e *customError) Error() string {
	return e.message
}
