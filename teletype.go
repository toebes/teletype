package main

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/tarm/serial"
)

var papertapeAlphabet = map[string]string{
	" ":    "\x00\x00",
	"\"":   "\x03\x00\x03",
	"#":    "\x0A\x1F\x0A\x1F\x0A",
	"$":    "\x12\x1D\x17\x09",
	"%":    "\x03\x1B\x04\x1B\x18",
	"&":    "\x0A\x15\x16\x08\x11",
	"'":    "\x03",
	"(":    "\x0E\x11",
	")":    "\x11\x0E",
	"*":    "\x0A\x04\x1F\x04\x0A",
	"+":    "\x04\x04\x1F\x04\x04",
	",":    "\x10\x08",
	"-":    "\x04\x04\x04",
	".":    "\x10",
	"/":    "\x18\x04\x03",
	"0":    "\x0E\x11\x11\x0E",
	"1":    "\x11\x1F\x10",
	"2":    "\x19\x15\x15\x12",
	"3":    "\x11\x11\x15\x15\x0A",
	"4":    "\x07\x04\x1F\x04",
	"5":    "\x17\x15\x15\x09",
	"6":    "\x0E\x15\x15\x08",
	"7":    "\x01\x19\x05\x03",
	"8":    "\x0A\x15\x15\x0A",
	"9":    "\x02\x15\x15\x0E",
	":":    "\x0A",
	" ;":   "\x10\x0A",
	" >":   "\x04\x0A\x11",
	" =":   "\x0A\x0A\x0A",
	" <":   "\x11\x0A\x04",
	" ?":   "\x01\x15\x05\x02",
	"@":    "\x0E\x11\x17\x15\x02",
	"A":    "\x1E\x05\x05\x1E",
	"B":    "\x1F\x15\x15\x0E",
	"C":    "\x0E\x11\x11\x0A",
	"D":    "\x1F\x11\x11\x0E",
	"E":    "\x1F\x15\x15\x15",
	"F":    "\x1F\x05\x05\x05",
	"G":    "\x0E\x11\x15\x0D",
	"H":    "\x1F\x04\x04\x1F",
	"I":    "\x1F",
	"J":    "\x08\x10\x10\x0F",
	"K":    "\x1F\x04\x0A\x11",
	"L":    "\x1F\x10\x10",
	"M":    "\x1F\x02\x04\x02\x1F",
	"N":    "\x1F\x02\x04\x08\x1F",
	"O":    "\x0E\x11\x11\x0E",
	"P":    "\x1F\x05\x05\x02",
	"Q":    "\x0E\x11\x09\x16",
	"R":    "\x1F\x05\x0D\x12",
	"S":    "\x12\x15\x15\x09",
	"T":    "\x01\x1F\x01",
	"U":    "\x0F\x10\x10\x0F",
	"V":    "\x03\x0C\x10\x0C\x03",
	"W":    "\x0F\x10\x0E\x10\x0F",
	"X":    "\x11\x0A\x04\x0A\x11",
	"Y":    "\x01\x02\x1C\x02\x01",
	"Z":    "\x19\x15\x13",
	" [":   "\x1F\x11",
	"\\":   "\x03\x04\x18",
	" ]":   "\x11\x1F",
	"^":    "\x02\x01\x02",
	" _":   "\x10\x10\x10",
	" `":   "\x01\x02",
	"a":    "\x1E\x05\x05\x1E",
	"b":    "\x1F\x15\x15\x0E",
	"c":    "\x0E\x11\x11\x0A",
	"d":    "\x1F\x11\x11\x0E",
	"e":    "\x1F\x15\x15\x15",
	"f":    "\x1F\x05\x05\x05",
	"g":    "\x0E\x11\x15\x0D",
	"h":    "\x1F\x04\x04\x1F",
	"i":    "\x1F",
	"j":    "\x08\x10\x10\x0F",
	"k":    "\x1F\x04\x0A\x11",
	"l":    "\x1F\x10\x10",
	"m":    "\x1F\x02\x04\x02\x1F",
	"n":    "\x1F\x02\x04\x08\x1F",
	"o":    "\x0E\x11\x11\x0E",
	"p":    "\x1F\x05\x05\x02",
	"q":    "\x0E\x11\x09\x16",
	"r":    "\x1F\x05\x0D\x12",
	"s":    "\x12\x15\x15\x09",
	"t":    "\x01\x1F\x01",
	"u":    "\x0F\x10\x10\x0F",
	"v":    "\x03\x0C\x10\x0C\x03",
	"w":    "\x0F\x10\x0E\x10\x0F",
	"x":    "\x11\x0A\x04\x0A\x11",
	"y":    "\x01\x02\x1C\x02\x01",
	"z":    "\x19\x15\x13",
	" {":   "\x04\x1B\x11",
	"|":    "\x1F",
	" }":   "\x11\x1B\x04",
	" ~":   "\x02\x01\x02\x01",
	"\x7f": "\x00\x00",
}

var basepath = "c:\\Users\\Timer1\\asciiart\\"

type mode int

var mission = 0

const (
	// Normal indicates that we are waiting for a command
	Normal mode = 1 + iota
	// GetName indicates that we are waiting for them to enter a name
	GetName
	// PrintTape indicates that we are waiting for return to start printing
	PrintTape
	// Exit indicates that we want to terminate the program
	Exit
)

type command int

const (
	// Stop indicates that the program should terminate cleanly
	Stop command = 1 + iota
	// Print indicates that we have data to output
	Print
	// Binary is like print, but carriage returns are not padded
	Binary
	// Read indicates that bytes have come in
	Read
)

type request struct {
	Command command
	Data    string
}

func printMission(writechan chan request) {
	output := ""
	switch mission {
	case 0:
		output = "Curiosity: Start at -140,60 and travel to 120,20 [Gale Crater]\n\r"
		mission = 1
	case 1:
		output = "Spirit: Start at -140,30 and travel to 170,-20\n\r"
		mission = 2
	case 2:
		output = "Viking 2: Start -140, 70 and travel to 140,40\n\r"
		mission = 0
	}
	writechan <- request{Print, "\n\r\n\n\n\n" + output + "\n\r\n\n\n\n\n\n\n\n\n\n\n\n\n>"}
}

func doCommand(command string, writechan chan request) mode {
	result := Normal
	command = strings.TrimSpace(command)
	fmt.Printf("Command: %s\n", command)
	switch command {
	case "?":
		writechan <- request{Print, "\n\rP - Print name on papertape\r\nM - Print a Mission\r\n>"}
	case "M":
		printMission(writechan)
	case "P":
		writechan <- request{Print, "\n\rType your name>"}
		result = GetName
	case "EXIT":
		writechan <- request{Print, "\n\rExiting\n\r"}
		result = Exit
	case "":
		writechan <- request{Print, "\n\r>"}
	default:
		file := basepath + strings.ToLower(strings.TrimSpace(command)) + ".txt"
		fmt.Printf("Attempting to open: %s\n", file)
		b, err := ioutil.ReadFile(file)
		if err == nil {
			// writechan <- request{Print, "\n\rFile not found: " + file + "\n\r"}
			out := string(b)
			writechan <- request{Print, "\n\r\n\n\n" + out + "\n\r\n\n\n\n"}
		}
		writechan <- request{Print, "\n\r>"}
	}
	return result
}

func printTape(text string, writechan chan request) {
	text = strings.Title(text)
	output := "\x00\x00\x00\x00\x00\x00\x00\x00\x00"
	for i := 0; i < len(text); i++ {
		txt, ok := papertapeAlphabet[string(text[i])]
		if ok {
			output += "\x00" + txt
		}
	}
	output += "\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"
	writechan <- request{Binary, output}
}
func main() {
	running := true
	curmode := Normal

	port := "COM3"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}
	c := &serial.Config{Name: port, Baud: 110, ReadTimeout: 100}
	readchan := make(chan request)
	writechan := make(chan request)
	var wg sync.WaitGroup
	wg.Add(2)
	s, err := serial.OpenPort(c)
	if err != nil {
		fmt.Printf("Attempted to open port %s\n", port)
		log.Fatal(err)
	}
	// We need to have two channels and the main thread
	// The first channel is responsible for writing to the serial port
	// It will receive either a request that says to terminate or to print to the port
	go func() {
		for running {
			req := <-writechan
			if req.Command == Print || req.Command == Binary {
				encodedStr := hex.EncodeToString([]byte(req.Data))
				fmt.Printf("Writing '%s'\n", encodedStr)
				for i := 0; i < len(req.Data); i++ {
					c := string(req.Data[i])
					// When we do a carriage return, we need to wait for the carriage to get all the way back
					if req.Command == Print && c == "\r" {
						c += "\x00\x00\x00"
					}
					_, err := s.Write([]byte(c))
					if err != nil {
						log.Fatal(err)
						break
					}
				}
			}
		}
		wg.Done()
	}()

	// The second channel is responsible for reading from the serial port.
	// As each character comes in, it sends it to the main thread.
	go func() {
		for running {
			buf := make([]byte, 4)
			n, err := s.Read(buf)
			if err != nil {
				log.Fatal(err)
				break
			}
			if n > 0 {
				// We need to strip the high bit off all the characters
				for i := 0; i < n; i++ {
					buf[i] &= 0x7f
				}
				readchan <- request{Read, string(buf[:n])}
			}
		}
		wg.Done()
	}()
	writechan <- request{Print, "\r\n\n\n\nSPACE MISSION CONTROL 2.0: READY FOR COMMANDS\r\n\n\n>"}
	command := ""
	name := ""
	for running {
		readreq := <-readchan
		switch readreq.Command {
		case Read:
			if readreq.Data == "\n" || readreq.Data == "\r" {
				switch curmode {
				case GetName:
					writechan <- request{Print, "\n\rPress On then Return>"}
					name = command
					command = ""
					curmode = PrintTape
				case PrintTape:
					printTape(name, writechan)
					curmode = Normal
				case Normal:
					curmode = doCommand(command, writechan)
				}
				if curmode == Exit {
					running = false
				}
				command = ""
			} else {
				encodedStr := hex.EncodeToString([]byte(readreq.Data))
				fmt.Printf("Read %s\n", encodedStr)
				if readreq.Data == "\x00" {
				} else if readreq.Data == "\x7f" {
					writechan <- request{Print, "\b"}
				} else {
					command += readreq.Data
					writechan <- request{Print, readreq.Data}
				}
			}
		}
	}
	wg.Wait()
}
