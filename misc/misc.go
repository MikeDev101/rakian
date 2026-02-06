package misc

import (
	"context"
	"fmt"
	"image"
	"log"
	"math"
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

var GlobalStorage map[string]any

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
		{103, 100 * time.Millisecond, 5}, // G7
		{91, 100 * time.Millisecond, 5},  // G6
		{0, time.Second, 1},              // NONE
	}

	player.Play(ctx, notes)
}

func PlayDeadBattery(player *tones.Tones, ctx context.Context) {
	notes := []tones.Note{
		{103, 100 * time.Millisecond, 5}, // G7
		{91, 100 * time.Millisecond, 5},  // G6
		{0, 200 * time.Millisecond, 1},   // NONE
		{103, 100 * time.Millisecond, 5}, // G7
		{91, 100 * time.Millisecond, 5},  // G6
		{0, 200 * time.Millisecond, 1},   // NONE
		{103, 100 * time.Millisecond, 5}, // G7
		{91, 100 * time.Millisecond, 5},  // G6
		{0, time.Second, 1},              // NONE
	}

	player.Play(ctx, notes)
}

func PlayRingtone(player *tones.Tones, ctx context.Context) {
	notes := []tones.Note{
		{88, 150 * time.Millisecond, 10}, // E7
		{86, 150 * time.Millisecond, 10}, // D#7 / Eb7
		{78, 300 * time.Millisecond, 10}, // G#6 / Ab6
		{80, 300 * time.Millisecond, 10}, // A#6 / Bb6
		{85, 150 * time.Millisecond, 10}, // D7
		{83, 150 * time.Millisecond, 10}, // C#7 / Db7
		{74, 300 * time.Millisecond, 10}, // D6
		{76, 300 * time.Millisecond, 10}, // E6
		{83, 150 * time.Millisecond, 10}, // C#7 / Db7
		{81, 150 * time.Millisecond, 10}, // B6
		{73, 300 * time.Millisecond, 10}, // C#6 / Db6
		{76, 300 * time.Millisecond, 10}, // E6
		{81, 600 * time.Millisecond, 10}, // B6
		{0, 3 * time.Second, 1},          // NONE
		{88, 150 * time.Millisecond, 2},  // E7
		{86, 150 * time.Millisecond, 2},  // D#7 / Eb7
		{78, 300 * time.Millisecond, 2},  // G#6 / Ab6
		{80, 300 * time.Millisecond, 2},  // A#6 / Bb6
		{85, 150 * time.Millisecond, 2},  // D7
		{83, 150 * time.Millisecond, 2},  // C#7 / Db7
		{74, 300 * time.Millisecond, 2},  // D6
		{76, 300 * time.Millisecond, 2},  // E6
		{83, 150 * time.Millisecond, 2},  // C#7 / Db7
		{81, 150 * time.Millisecond, 2},  // B6
		{73, 300 * time.Millisecond, 2},  // C#6 / Db6
		{76, 300 * time.Millisecond, 2},  // E6
		{81, 600 * time.Millisecond, 2},  // B6
		{0, 3 * time.Second, 1},          // NONE
	}

	player.Play(ctx, notes)
}

func VibrateAlert(player *tones.Tones, ctx context.Context) {
	states := []tones.Vibrate{
		{true, 300 * time.Millisecond},
		{false, 100 * time.Millisecond},
		{true, 300 * time.Millisecond},
		{false, 100 * time.Millisecond},
	}
	player.Vibrate(ctx, states)
}

func StartVibrate(player *tones.Tones, ctx context.Context) {
	var states []tones.Vibrate
	for range 3 {
		for _, elem := range []tones.Vibrate{
			{true, 200 * time.Millisecond},
			{false, 200 * time.Millisecond},
			{true, 200 * time.Millisecond},
			{false, 200 * time.Millisecond},
			{true, 500 * time.Millisecond},
			{false, 500 * time.Millisecond},
		} {
			states = append(states, elem)
		}
	}
	player.Vibrate(ctx, states)
}

func PlayBeep(player *tones.Tones, ctx context.Context) {
	notes := []tones.Note{
		{88, 150 * time.Millisecond, 2}, // E7
		{0, 20 * time.Millisecond, 1},   // NONE
		{88, 300 * time.Millisecond, 2}, // E7
		{0, 3 * time.Second, 1},         // NONE
	}
	player.Play(ctx, notes)
}

func PlayBoot(player *tones.Tones, ctx context.Context) {
	offset := 9
	notes := []tones.Note{
		{83 + offset, 300 * time.Millisecond, 10},  // C#7 / Db7
		{81 + offset, 300 * time.Millisecond, 10},  // B6
		{73 + offset, 600 * time.Millisecond, 10},  // C#6 / Db6
		{76 + offset, 600 * time.Millisecond, 10},  // E6
		{81 + offset, 1200 * time.Millisecond, 10}, // B6
	}

	player.Play(ctx, notes)
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

	// Scale values
	voltage = float64(voltageRaw) / 1000000.0
	capacity_scaled = int(math.Round(float64(capacity) / 10.0))

	return voltage, capacity, capacity_scaled, nil
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
