package misc

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"tones"
)

func EnablePowerbutton() {
	cmd := exec.Command("enable_poweroff")
	err := cmd.Run()
	if err != nil {
		log.Println("Failed to enable the powerbutton:", err)
	} else {
		log.Println("Hardware power button enabled.")
	}
}

func DisablePowerbutton() {
	cmd := exec.Command("disable_poweroff")
	err := cmd.Run()
	if err != nil {
		log.Println("Failed to disable the powerbutton:", err)
	} else {
		log.Println("Hardware power button disabled.")
	}
}

func Shutdown() {
	cmd := exec.Command("poweroff")
	err := cmd.Run()
	if err != nil {
		log.Println("Failed to shutdown:", err)
	} else {
		log.Println("Shutdown command issued.")
	}
	os.Exit(0)
}

func HardReboot() {
	cmd := exec.Command("reboot", "now")
	err := cmd.Run()
	if err != nil {
		log.Println("Failed to hard reboot:", err)
	} else {
		log.Println("Hard reboot command issued.")
	}
	os.Exit(0)
}

func SoftReboot() {
	cmd := exec.Command("systemctl", "restart", "rakian")
	err := cmd.Run()
	if err != nil {
		log.Println("Failed to soft reboot:", err)
	} else {
		log.Println("Soft reboot command issued.")
	}
	os.Exit(0)
}

func KeyLightsOn() {
	/* if err := rpio.Open(); err != nil {
		panic(err)
	}
	p := rpio.Pin(13)
	p.Output()
	p.High() */
}

func KeyLightsOff() {
	/* if err := rpio.Open(); err != nil {
		panic(err)
	}
	p := rpio.Pin(13)
	p.Output()
	p.Low() */
}

func SleepWithContext(duration time.Duration, ctx context.Context) {
	timer := time.NewTimer(duration)
	select {
	case <-ctx.Done():
		timer.Stop()
		return
	case <-timer.C:
	}
}

func PlayLowBattery(player *tones.Tones, ctx context.Context) {
	notes := []tones.Note{
		{Key: 103, Duration: 100 * time.Millisecond, Divider: 5}, // G7
		{Key: 91, Duration: 100 * time.Millisecond, Divider: 5},  // G6
		{Key: 0, Duration: time.Second, Divider: 1},              // NONE
	}

	player.Play(ctx, notes)
}

func PlayDeadBattery(player *tones.Tones, ctx context.Context) {
	notes := []tones.Note{
		{Key: 103, Duration: 100 * time.Millisecond, Divider: 5}, // G7
		{Key: 91, Duration: 100 * time.Millisecond, Divider: 5},  // G6
		{Key: 0, Duration: 200 * time.Millisecond, Divider: 1},   // NONE
		{Key: 103, Duration: 100 * time.Millisecond, Divider: 5}, // G7
		{Key: 91, Duration: 100 * time.Millisecond, Divider: 5},  // G6
		{Key: 0, Duration: 200 * time.Millisecond, Divider: 1},   // NONE
		{Key: 103, Duration: 100 * time.Millisecond, Divider: 5}, // G7
		{Key: 91, Duration: 100 * time.Millisecond, Divider: 5},  // G6
		{Key: 0, Duration: time.Second, Divider: 1},              // NONE
	}

	player.Play(ctx, notes)
}

func PlayRingtone(player *tones.Tones, ctx context.Context) {
	notes := []tones.Note{
		{Key: 88, Duration: 150 * time.Millisecond, Divider: 1}, // E7
		{Key: 86, Duration: 150 * time.Millisecond, Divider: 1}, // D#7 / Eb7
		{Key: 78, Duration: 300 * time.Millisecond, Divider: 1}, // G#6 / Ab6
		{Key: 80, Duration: 300 * time.Millisecond, Divider: 1}, // A#6 / Bb6
		{Key: 85, Duration: 150 * time.Millisecond, Divider: 1}, // D7
		{Key: 83, Duration: 150 * time.Millisecond, Divider: 1}, // C#7 / Db7
		{Key: 74, Duration: 300 * time.Millisecond, Divider: 1}, // D6
		{Key: 76, Duration: 300 * time.Millisecond, Divider: 1}, // E6
		{Key: 83, Duration: 150 * time.Millisecond, Divider: 1}, // C#7 / Db7
		{Key: 81, Duration: 150 * time.Millisecond, Divider: 1}, // B6
		{Key: 73, Duration: 300 * time.Millisecond, Divider: 1}, // C#6 / Db6
		{Key: 76, Duration: 300 * time.Millisecond, Divider: 1}, // E6
		{Key: 81, Duration: 600 * time.Millisecond, Divider: 1}, // B6
		{Key: 0, Duration: 3 * time.Second, Divider: 1},         // NONE
	}

	player.Play(ctx, notes)
}

func PlayBeep(player *tones.Tones, ctx context.Context) {
	notes := []tones.Note{
		{Key: 88, Duration: 150 * time.Millisecond, Divider: 2}, // E7
		{Key: 0, Duration: 20 * time.Millisecond, Divider: 1},   // NONE
		{Key: 88, Duration: 300 * time.Millisecond, Divider: 2}, // E7
		{Key: 0, Duration: 3 * time.Second, Divider: 1},         // NONE
	}
	player.Play(ctx, notes)
}

func PlayBoot(player *tones.Tones, ctx context.Context) {
	offset := 9.0
	notes := []tones.Note{
		{Key: 83 + offset, Duration: 300 * time.Millisecond, Divider: 1}, // C#7 / Db7
		{Key: 81 + offset, Duration: 300 * time.Millisecond, Divider: 1}, // B6
		{Key: 73 + offset, Duration: 500 * time.Millisecond, Divider: 1}, // C#6 / Db6
		{Key: 76 + offset, Duration: 500 * time.Millisecond, Divider: 1}, // E6
		{Key: 81 + offset, Duration: 750 * time.Millisecond, Divider: 1}, // B6
	}

	player.Play(ctx, notes)
}

func GetChargingStatus() (charging bool) {
	// Read status
	state, err := os.ReadFile("/sys/class/power_supply/charger/online")
	if err != nil {
		return false
	}
	charging = strings.TrimSpace(string(state)) == "1"
	return charging
}

func GetBatteryStatus() (voltage float64, capacity int, capacity_scaled int, err error) {
	// Read capacity
	capacityBytes, err := os.ReadFile("/sys/class/power_supply/battery/capacity")
	if err != nil {
		return 0.0, 0, 0, fmt.Errorf("reading capacity failed: %w", err)
	}
	capacityStr := strings.TrimSpace(string(capacityBytes))
	capacity, err = strconv.Atoi(capacityStr)
	if err != nil {
		return 0.0, 0, 0, fmt.Errorf("converting capacity to int failed: %w", err)
	}

	// Read voltage
	voltageBytes, err := os.ReadFile("/sys/class/power_supply/battery/voltage_now")
	if err != nil {
		return 0.0, 0, 0, fmt.Errorf("reading voltage failed: %w", err)
	}
	voltageStr := strings.TrimSpace(string(voltageBytes))
	voltageRaw, err := strconv.Atoi(voltageStr)
	if err != nil {
		return 0.0, 0, 0, fmt.Errorf("converting raw voltage to int failed: %w", err)
	}

	// Cap capacity at 100%
	if capacity > 100 {
		capacity = 100
	}

	// Scale values
	voltage = float64(voltageRaw) / 1000000.0

	// Scale to 0–4
	capacity_scaled = int(math.Round((float64(capacity) / 100.0) * 4.0))

	return voltage, capacity, capacity_scaled, nil
}

func GetOSVersion() string {
	output, err := exec.Command("awk", "-F=", "$1==\"PRETTY_NAME\" { print $2 ;}", "/etc/os-release").CombinedOutput()
	if err != nil {
		panic(fmt.Errorf("os-release failed: %w", err))
	}
	return strings.ReplaceAll(strings.TrimSpace(string(output)), "\"", "")
}

func CheckCellularData(apn string) bool {
	cmd := exec.Command("/usr/bin/cellular_toggle", "status", apn)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error checking cellular: %v", err)
	}
	enabled := strings.TrimSpace(string(output)) == "1"
	return enabled
}

func SetCellularDataState(apn string, state bool) bool {
	mode := "disable"
	if state {
		mode = "enable"
	}

	cmd := exec.Command("/usr/bin/cellular_toggle", mode, apn)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error setting cellular: %v", err)
	}
	enabled := strings.TrimSpace(string(output)) == "1"
	return enabled
}

func ToggleCellularData(apn string) bool {
	cmd := exec.Command("/usr/bin/cellular_toggle", "toggle", apn)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error toggling cellular: %v", err)
	}
	enabled := strings.TrimSpace(string(output)) == "1"
	return enabled
}

func GetWiFiStatus() (connected bool, ssid string, signalScaled int, ipaddress string) {
	output, err := exec.Command("iwconfig").CombinedOutput()
	if err != nil {
		panic(fmt.Errorf("iwconfig failed: %w", err))
	}

	outStr := string(output)

	// Check if we have a valid ESSID
	reSSID := regexp.MustCompile(`ESSID:"([^"]+)"`)
	ssidMatch := reSSID.FindStringSubmatch(outStr)
	if len(ssidMatch) > 1 && ssidMatch[1] != "off/any" {
		ssid = ssidMatch[1]
		connected = true
	} else {
		return false, "", 0, "" // Not connected
	}

	// Grab signal level in dBm
	reSignal := regexp.MustCompile(`Signal level=(-?\d+) dBm`)
	signalMatch := reSignal.FindStringSubmatch(outStr)
	if len(signalMatch) > 1 {
		dBm, _ := strconv.Atoi(signalMatch[1])

		// Convert dBm to rough percentage (from -100 to -50)
		percent := 2 * (dBm + 100)
		if percent < 0 {
			percent = 0
		} else if percent > 100 {
			percent = 100
		}

		// Scale to 0–5
		signalScaled = percent * 6 / 100
		if signalScaled > 5 {
			signalScaled = 5
		} else if signalScaled < 0 {
			signalScaled = 0
		}
	}

	// Get IP address
	ipString, err := exec.Command("nmcli", "-g", "IP4.ADDRESS", "device", "show", "wlan0").Output()
	if err != nil {
		return connected, ssid, signalScaled, ""
	}

	ipaddress = string(ipString)

	return connected, ssid, signalScaled, ipaddress
}

func GetModemStatusMMCLI() (state string, operator string, signal string) {
	// Use mmcli to get modem status in key-value format
	out, err := exec.Command("mmcli", "-m", "any", "-K").Output()
	if err != nil {
		return "error", "", "0"
	}

	lines := strings.SplitSeq(string(out), "\n")
	for line := range lines {
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) < 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "modem.status.state":
			state = val
		case "modem.3gpp.operator-name":
			operator = val
		case "modem.status.signal-quality.value":
			signal = val
		}
	}
	return state, operator, signal
}

// CheckConnectivity attempts to GET a lightweight URL.
// It returns true if the status code is 204 (No Content),
// which is the standard response for this Google endpoint.
func CheckConnectivity(ctx context.Context) bool {
	log.Println("🌎 Checking connectivity...")

	// We use a short timeout for the request itself so the
	// function doesn't hang if the network is "blackholed."
	reqCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	url := "http://connectivitycheck.gstatic.com/generate_204"

	req, err := http.NewRequestWithContext(reqCtx, "GET", url, nil)
	if err != nil {
		log.Printf("🌎 Connectivity check failure: %v", err)
		return false
	}

	// Using DefaultClient; for production, consider a custom client
	// with specific Transport settings.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	res := resp.StatusCode == http.StatusNoContent

	if res {
		log.Println("🌎 Connectivity check success")
	} else {
		log.Printf("🌎 Connectivity check failure: %s", resp.Status)
	}

	return res
}

func IsBluetoothEnabled() bool {
	out, err := exec.Command("bluetoothctl", "show").Output()
	if err != nil {
		log.Println("⚠️ Failed to check bluetooth status:", err)
		return false
	}
	return strings.Contains(string(out), "Powered: yes")
}
