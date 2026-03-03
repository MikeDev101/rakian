package main

import (
	"db"

	"github.com/Wifx/gonetworkmanager/v3"
	"github.com/glebarez/sqlite"
	"github.com/godbus/dbus/v5"
	"gorm.io/gorm"
)

func InitNetworkManager() (gonetworkmanager.NetworkManager, gonetworkmanager.DeviceWireless) {
	nm, err := gonetworkmanager.NewNetworkManager()
	if err != nil {
		panic(err)
	}

	devices, err := nm.GetDevices()
	if err != nil {
		panic(err)
	}

	var wifi_device_raw dbus.ObjectPath
	for _, device := range devices {
		device_interface, err := device.GetPropertyInterface()
		if err != nil {
			panic(err)
		}
		if device_interface == WLAN_DEVICE {
			wifi_device_raw = device.GetPath()
			break
		}
	}

	if wifi_device_raw == "" {
		panic("No wifi device found")
	}

	wifi_device, err := gonetworkmanager.NewDeviceWireless(wifi_device_raw)
	if err != nil {
		panic(err)
	}

	// Failsafe
	enabled, _ := nm.GetPropertyWirelessEnabled()
	if !enabled {
		nm.SetPropertyWirelessEnabled(true)
	}

	return nm, wifi_device
}

func InitDB() *gorm.DB {
	database, err := gorm.Open(sqlite.Open(DB_PATH), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	database.AutoMigrate(&db.KVStore{})
	database.AutoMigrate(&db.Contacts{})
	database.AutoMigrate(&db.Messages{})
	database.AutoMigrate(&db.CallSessionEvents{})
	database.AutoMigrate(&db.CallStateEvents{})
	return database
}
