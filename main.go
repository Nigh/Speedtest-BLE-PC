package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"sort"
	"time"

	"tinygo.org/x/bluetooth"
)

var (
	adapter = bluetooth.DefaultAdapter

	speedtestServiceUUID = bluetooth.New16BitUUID(0xA800)
	notifyCharUUID       = bluetooth.New16BitUUID(0xA801)
	writeCharUUID        = bluetooth.New16BitUUID(0xA802)
)

var packageWindow []int
var packageRepeat []int
var packageLoss []int

func init() {
	packageWindow = make([]int, 0)
	packageRepeat = make([]int, 0)
	packageLoss = make([]int, 0)
}

func packageIdxPush(idx int) {
	packageWindow = append(packageWindow, idx)
	sort.Ints(packageWindow)
	// remove duplicates
	for i := 0; i < len(packageWindow)-1; i++ {
		if packageWindow[i] == packageWindow[i+1] {
			packageRepeat = append(packageRepeat, packageWindow[i])
			packageWindow = append(packageWindow[:i], packageWindow[i+1:]...)
			i--
		}
	}
}

func packageContinuous() (res bool, loss int) {
	loss = -1
	res = true
	if len(packageWindow) < 8 {
		return
	}
	for i := 0; i < len(packageWindow)-2; i++ {
		if packageWindow[i+1]-packageWindow[i] != 1 {
			loss = packageWindow[i] + 1
			res = false
			packageWindow = packageWindow[i+1:]
			packageIdxPush(loss)
			return
		}
	}
	for len(packageWindow) > 8 {
		packageWindow = packageWindow[1:]
	}
	return
}

func main() {
	println("enabling")

	// Enable BLE interface.
	must("enable BLE stack", adapter.Enable())

	ch := make(chan bluetooth.ScanResult, 1)

	// Start scanning.
	println("scanning...")
	err := adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
		isValidDevice := func(result bluetooth.ScanResult) bool {
			md := result.ManufacturerData()
			for _, v := range md {
				if v.CompanyID == 0x5EE5 {
					return true
				}
			}
			return false
		}
		if result.LocalName() == "Speedtest" && isValidDevice(result) {
			println("found device:", result.Address.String(), result.LocalName(), result.RSSI)
			if result.RSSI > -75 {
				adapter.StopScan()
				ch <- result
			}
		}
	})

	var device bluetooth.Device
	select {
	case result := <-ch:
		device, err = adapter.Connect(result.Address, bluetooth.ConnectionParams{})
		if err != nil {
			println(err.Error())
			return
		}
		println("connected to ", result.Address.String())
	}

	// get services
	println("discovering services/characteristics")
	srvcs, err := device.DiscoverServices([]bluetooth.UUID{speedtestServiceUUID})
	must("discover services", err)

	if len(srvcs) == 0 {
		panic("could not find heart rate service")
	}

	srvc := srvcs[0]

	println("found service", srvc.UUID().String())
	chars, err := srvc.DiscoverCharacteristics([]bluetooth.UUID{notifyCharUUID, writeCharUUID})
	if err != nil {
		println(err)
	}

	if len(chars) < 2 {
		panic("could not find speedtest characteristic")
	}

	notifyChar := chars[0]
	writeChar := chars[1]
	println("found characteristic", notifyChar.UUID().String(), writeChar.UUID().String())

	TestPackage := make(chan []byte, 1)
	startTime := time.Now()
	lengthRecieved := 0
	packageReceived := 0

	notifyChar.EnableNotifications(func(buf []byte) {
		TestPackage <- buf
	})

	go func() {
		println("Test will start after 3 seconds")
		for i := 0; i < 3; i++ {
			<-time.After(1 * time.Second)
			println(3 - i)
		}

		go func() {
			for {
				select {
				case data := <-TestPackage:
					idx := binary.LittleEndian.Uint32(data[0:4])
					fmt.Printf("package:%5d length:%d\n", idx, len(data))
					packageIdxPush(int(idx))
					good, loss := packageContinuous()
					if good {
						packageReceived += 1
						lengthRecieved += len(data)
					} else {
						for !good {
							packageLoss = append(packageLoss, loss)
							good, loss = packageContinuous()
						}
					}
				case <-time.After(1 * time.Second):
					println("Test complete")
					duration := time.Since(startTime.Add(1 * time.Second))
					fmt.Printf("Transfer %d packages(%d bytes) in %dms\n", packageReceived, lengthRecieved, duration.Milliseconds())
					fmt.Printf("%d packages loss %v\n", len(packageLoss), packageLoss)
					fmt.Printf("%d packages repeat %v\n", len(packageRepeat), packageRepeat)
					fmt.Printf("Average speed is %.2f kB/s\n", float64(lengthRecieved)/1000.0/duration.Seconds())
					os.Exit(0)
				}
			}
		}()

		writeChar.WriteWithoutResponse([]byte{0x01, 0xAA, 0x03, 0x00})
		startTime = time.Now()
		lengthRecieved = 0
		packageReceived = 0
		packageRepeat = make([]int, 0)
		packageLoss = make([]int, 0)
	}()

	select {}
}

func must(action string, err error) {
	if err != nil {
		panic("failed to " + action + ": " + err.Error())
	}
}
