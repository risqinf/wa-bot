package utils

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorCyan   = "\033[36m"
)

func LogRealtime(sender, target, msgType, content string, isMe bool) {
	timestamp := time.Now().Format("15:04:05")
	if isMe {
		fmt.Printf("%s[%s] %s[ME] %s-> %s%s [%s]: %s\n",
			ColorBlue, timestamp, ColorGreen, ColorReset, ColorYellow, target, msgType, content)
	} else {
		fmt.Printf("%s[%s] %s[IN] %s%s %s-> Bot [%s]: %s\n",
			ColorBlue, timestamp, ColorCyan, ColorReset, sender, ColorReset, msgType, content)
	}
}

func LogCommand(sender, command string, duration time.Duration) {
	timestamp := time.Now().Format("15:04:05")
	user := strings.Split(sender, "@")[0]
	fmt.Printf("%s[%s]%s %s[CMD]%s %s%-15s%s -> %s%s%s (%.3fs)\n",
		ColorBlue, timestamp, ColorReset, ColorYellow, ColorReset,
		ColorGreen, user, ColorReset, ColorRed, command, ColorReset, duration.Seconds())
}

func LogDebug(msg string) {
	fmt.Printf("%s[DEBUG] %s%s\n", ColorRed, msg, ColorReset)
}

func FormatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	if days > 0 {
		return fmt.Sprintf("%d hari %d jam", days, hours)
	}
	return fmt.Sprintf("%d jam %d menit", hours, minutes)
}

func RandomDelay() {
	ms := 1000 + rand.Intn(2000)
	time.Sleep(time.Duration(ms) * time.Millisecond)
}
