// Copyright 2017 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Methods for interfacing with the field PLC.

package serial

import (
	"fmt"
	"log"
	"strings"

	"go.bug.st/serial"
)

var port = ""

type Arduino struct {
	IsHealthy  bool
	fieldEstop bool
}

// Returns true if the Arduino is enabled in the configurations.
func (arduino *Arduino) IsEnabled() bool {
	return true
}

func (arduino *Arduino) Run() {
	arduino.fieldEstop = false

	mode := &serial.Mode{
		BaudRate: 9600,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}

	// Retrieve the port list
	ports, err := serial.GetPortsList()
	if err != nil {
		log.Printf("Error retrieving port list: %v", err)
	}

	// Print the list of detected ports
	for _, port1 := range ports {
		fmt.Printf("Found port: %v\n", port1)

		port, err := serial.Open(port1, mode)
		if err != nil {
			log.Fatal(err)
		}

		// Wait for "pong" response
		buf := make([]byte, 100)
		str := ""
		for {
			n, err := port.Read(buf)
			fmt.Println(n)
			str += string(buf[:n])
			if err != nil {
				log.Printf("Error reading pong: %v", err)
			} else if strings.Contains(str, "15") {
				arduino.IsHealthy = true
				fmt.Printf("Received pong from Arduino (%v)", port1)
				break
			} else {
				arduino.IsHealthy = false
				fmt.Printf("Unexpected response from Arduino: %s\n", string(buf[:n]))
			}
		}

		if !arduino.IsHealthy {
			continue
		}

		str = ""

		for {
			n, err := port.Read(buf)

			if err != nil {
				log.Printf("Error reading from port: %v", err)
			}

			if string(buf[:n]) != "9" {
				arduino.fieldEstop = false
				continue
			}
			arduino.fieldEstop = true
		}

		// Log the status of receivedPong
	}
}

func (arduino *Arduino) GetFieldEstop() bool {
	return arduino.fieldEstop
}
