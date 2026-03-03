package main

// go build -ldflags "-X 'main.DEBUG_MODE=false'" .
var DEBUG_MODE string = "true"
var FW_VERSION string = "0.1.20 (3.2.2026)"
var EXIT_MODE uint8 = 0        // 0 - none, 1 - shutdown, 2 - reboot, 3 - soft restart
const WLAN_DEVICE = "wlp2s0"   // wlan0
const DB_PATH = "./kvstore.db" // "/root/rakian/kvstore.db"
const (
	EXIT_SHUTDOWN = 1
	EXIT_REBOOT   = 2
	EXIT_RESTART  = 3
)
