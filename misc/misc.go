package misc

import (
	"context"
	"fmt"
	"image"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"sh1107"
	"tones"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/host/v3"
)

func Shutdown() {
	cmd := exec.Command("poweroff")
	err := cmd.Run()
	if err != nil {
		log.Println("Failed to shutdown:", err)
	} else {
		log.Println("Shutdown command issued.")
	}
}

func HardReboot() {
	cmd := exec.Command("reboot", "now")
	err := cmd.Run()
	if err != nil {
		log.Println("Failed to hard reboot:", err)
	} else {
		log.Println("Hard reboot command issued.")
	}
}

func SoftReboot() {
	cmd := exec.Command("systemctl", "restart", "rakian")
	err := cmd.Run()
	if err != nil {
		log.Println("Failed to soft reboot:", err)
	} else {
		log.Println("Soft reboot command issued.")
	}
}

func KeyLightsOn() {
	if _, err := host.Init(); err != nil {
		panic(err)
	}
	p := gpioreg.ByName("GPIO23")
	if p == nil {
		log.Fatal("Failed to find GPIO23 (Keypad light control)")
	}

	if err := p.Out(gpio.High); err != nil {
		log.Fatal(err)
	}
}

func KeyLightsOff() {
	if _, err := host.Init(); err != nil {
		panic(err)
	}
	p := gpioreg.ByName("GPIO23")
	if p == nil {
		log.Fatal("Failed to find GPIO23 (Keypad light control)")
	}

	if err := p.Out(gpio.Low); err != nil {
		log.Fatal(err)
	}
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
		{Key: 88, Duration: 150 * time.Millisecond, Divider: 10}, // E7
		{Key: 86, Duration: 150 * time.Millisecond, Divider: 10}, // D#7 / Eb7
		{Key: 78, Duration: 300 * time.Millisecond, Divider: 10}, // G#6 / Ab6
		{Key: 80, Duration: 300 * time.Millisecond, Divider: 10}, // A#6 / Bb6
		{Key: 85, Duration: 150 * time.Millisecond, Divider: 10}, // D7
		{Key: 83, Duration: 150 * time.Millisecond, Divider: 10}, // C#7 / Db7
		{Key: 74, Duration: 300 * time.Millisecond, Divider: 10}, // D6
		{Key: 76, Duration: 300 * time.Millisecond, Divider: 10}, // E6
		{Key: 83, Duration: 150 * time.Millisecond, Divider: 10}, // C#7 / Db7
		{Key: 81, Duration: 150 * time.Millisecond, Divider: 10}, // B6
		{Key: 73, Duration: 300 * time.Millisecond, Divider: 10}, // C#6 / Db6
		{Key: 76, Duration: 300 * time.Millisecond, Divider: 10}, // E6
		{Key: 81, Duration: 600 * time.Millisecond, Divider: 10}, // B6
		{Key: 0, Duration: 3 * time.Second, Divider: 1},          // NONE
		{Key: 88, Duration: 150 * time.Millisecond, Divider: 2},  // E7
		{Key: 86, Duration: 150 * time.Millisecond, Divider: 2},  // D#7 / Eb7
		{Key: 78, Duration: 300 * time.Millisecond, Divider: 2},  // G#6 / Ab6
		{Key: 80, Duration: 300 * time.Millisecond, Divider: 2},  // A#6 / Bb6
		{Key: 85, Duration: 150 * time.Millisecond, Divider: 2},  // D7
		{Key: 83, Duration: 150 * time.Millisecond, Divider: 2},  // C#7 / Db7
		{Key: 74, Duration: 300 * time.Millisecond, Divider: 2},  // D6
		{Key: 76, Duration: 300 * time.Millisecond, Divider: 2},  // E6
		{Key: 83, Duration: 150 * time.Millisecond, Divider: 2},  // C#7 / Db7
		{Key: 81, Duration: 150 * time.Millisecond, Divider: 2},  // B6
		{Key: 73, Duration: 300 * time.Millisecond, Divider: 2},  // C#6 / Db6
		{Key: 76, Duration: 300 * time.Millisecond, Divider: 2},  // E6
		{Key: 81, Duration: 600 * time.Millisecond, Divider: 2},  // B6
		{Key: 0, Duration: 3 * time.Second, Divider: 1},          // NONE
	}

	player.Play(ctx, notes)
}

func VibrateAlert(player *tones.Tones, ctx context.Context) {
	states := []tones.Vibrate{
		{State: true, Duration: 300 * time.Millisecond},
		{State: false, Duration: 100 * time.Millisecond},
		{State: true, Duration: 300 * time.Millisecond},
		{State: false, Duration: 100 * time.Millisecond},
	}
	player.Vibrate(ctx, states)
}

func StartVibrate(player *tones.Tones, ctx context.Context) {
	var states []tones.Vibrate
	for range 3 {
		for _, elem := range []tones.Vibrate{
			{State: true, Duration: 200 * time.Millisecond},
			{State: false, Duration: 200 * time.Millisecond},
			{State: true, Duration: 200 * time.Millisecond},
			{State: false, Duration: 200 * time.Millisecond},
			{State: true, Duration: 500 * time.Millisecond},
			{State: false, Duration: 500 * time.Millisecond},
		} {
			states = append(states, elem)
		}
	}
	player.Vibrate(ctx, states)
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
	offset := 9
	notes := []tones.Note{
		{Key: 83 + offset, Duration: 300 * time.Millisecond, Divider: 10},  // C#7 / Db7
		{Key: 81 + offset, Duration: 300 * time.Millisecond, Divider: 10},  // B6
		{Key: 73 + offset, Duration: 600 * time.Millisecond, Divider: 10},  // C#6 / Db6
		{Key: 76 + offset, Duration: 600 * time.Millisecond, Divider: 10},  // E6
		{Key: 81 + offset, Duration: 1200 * time.Millisecond, Divider: 10}, // B6
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
	capacity_scaled = int(math.Round(float64(capacity) / 10.0))

	return voltage, capacity, capacity_scaled, nil
}

func GetOSVersion() string {
	output, err := exec.Command("awk", "-F=", "'$1==\"PRETTY_NAME\"", "{ print $2 ;}'", "/etc/os-release").CombinedOutput()
	if err != nil {
		panic(fmt.Errorf("os-release failed: %w", err))
	}
	return strings.ReplaceAll(strings.TrimSpace(string(output)), "\"", "")
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

		// Scale to 0–7
		signalScaled = percent * 8 / 100
		if signalScaled > 7 {
			signalScaled = 7
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

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
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

func SwitchToPowerSaveMode() {
	err := exec.Command("cpupower", "frequency-set", "--governor", "powersave").Run()
	if err != nil {
		panic(err)
	}
}

func SwitchToNormalMode() {
	err := exec.Command("cpupower", "frequency-set", "--governor", "schedutil").Run()
	if err != nil {
		panic(err)
	}
}

func TestText(display *sh1107.SH1107) {
	log.Print("Testing lowercase")

	fonts := []struct {
		name     string
		set      func() map[rune]image.Image
		row_next int
	}{
		{
			"8 (Thin)",
			display.Use_Font8_Normal,
			12,
		},
		{
			"8 (Bold)",
			display.Use_Font8_Bold,
			12,
		},
		{
			"16",
			display.Use_Font16,
			18,
		},
	}

	for _, font_entry := range fonts {

		font := font_entry.set()
		tests := [][]string{
			{
				"lowercase",
				"the quick brown",
				"fox jumps over",
				"the lazy dog",
			},
			{
				"UPPERCASE",
				"THE QUICK BROWN",
				"FOX JUMPS OVER",
				"THE LAZY DOG",
			},
			{
				"MixedCase",
				"The Quick Brown",
				"Fox Jumps Over",
				"The Lazy Dog",
			},
			{
				"Symbols",
				"1234567890",
				"~`!@#$%^&*()€",
				"_+-=[]|\\:;'",
				"\"<>,./?£¤¥§",
			},
		}

		// Test all cases
		for _, test_entry := range tests {

			log.Printf("Testing %s using font %s", test_entry[0], font_entry.name)

			display.Clear(sh1107.Black)
			for i := range len(test_entry) - 1 {
				display.DrawText(0, 20+(font_entry.row_next*(i+1)), font, test_entry[i+1], false)
			}
			display.Render()
			time.Sleep(3 * time.Second)
		}
	}

	log.Println("Testing time font")
	font := display.Use_Font_Time()
	display.Clear(sh1107.Black)
	display.DrawText(0, 20, font, "1234567890", false)
	display.DrawText(0, 32, font, "/.-", false)
	display.Render()
	time.Sleep(3 * time.Second)
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
