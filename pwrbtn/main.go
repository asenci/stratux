package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/stianeikeland/go-rpio/v4"
	"github.com/takama/daemon"
)

const (
	name        = "pwrbtn"
	description = "monitor power button presses"

	defaultPin = 22
)

func handleCommand(service daemon.Daemon, command string) (string, error) {
	switch command {
	case "install":
		return service.Install()
	case "remove":
		return service.Remove()
	case "start":
		return service.Start()
	case "stop":
		return service.Stop()
	case "status":
		return service.Status()
	default:
		return "Usage: " + name + " install | remove | start | stop | status", nil
	}
}

func main() {
	btnPin := flag.Int("pin", defaultPin, "power button pin (BCM numbering)")
	flag.Parse()

	service, err := daemon.New(name, description, daemon.SystemDaemon)
	if err != nil {
		log.Fatal(err)
	}

	// if received any kind of command, do it
	if flag.NArg() > 0 {
		command := flag.Args()[0]
		status, err := handleCommand(service, command)
		if err != nil {
			log.Fatal(status, "\nError: ", err)
		}
		fmt.Println(status)
		return
	}

	if err := rpio.Open(); err != nil {
		log.Fatal(err)
	}
	defer rpio.Close()

	pin := rpio.Pin(*btnPin)
	pin.Input()
	pin.PullUp()
	pin.Detect(rpio.AnyEdge)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, os.Kill, syscall.SIGTERM)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case killSignal := <-interrupt:
			fmt.Println("Got signal:", killSignal)
			return
		case <-ticker.C:
			if pin.EdgeDetected() {
				fmt.Println("Button press detected, shutting down...")
				syscall.Sync()
				if err := exec.Command("systemctl", "poweroff").Run(); err != nil {
					log.Println(err)
				}
			}
		}
	}
}
