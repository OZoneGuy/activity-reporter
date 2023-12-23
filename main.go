package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/screensaver"
	"github.com/jezek/xgb/xproto"
)

func main() {

	// setup MQTT client

	var (
		MQTT_BROKER = os.Getenv("MQTT_BROKER")
		MQTT_USER   = os.Getenv("MQTT_USER")
		MQTT_PASS   = os.Getenv("MQTT_PASS")
	)

	opts := mqtt.NewClientOptions().AddBroker(MQTT_BROKER)
	opts.SetClientID("Linux-PC-Monitor")
	opts.SetUsername(MQTT_USER)
	opts.SetPassword(MQTT_PASS)

	client := mqtt.NewClient(opts)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Println("Failed to connect to MQTT broker")
		panic(token.Error())
	}

	signalChannel := make(chan os.Signal, 2)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		sig := <-signalChannel
		client.Publish("homeassistant/sensor/BIG-DISK-ENERGY/PC_Monitor/state", 0, false, "PowerOff")
		client.Publish("homeassistant/sensor/BIG-DISK-ENERGY/availability", 0, false, "offline")
		client.Disconnect(250)
		fmt.Println(time.Now())
		fmt.Println(sig)
		fmt.Println("Exiting")
		os.Exit(0)
	}()

	var err error
	var isIdle bool
	// idle.Get()
	for err == nil {
		isIdle, err = isInactive()

		var value string
		if !isIdle {
			value = "PowerOn"
		} else {
			value = "PowerOff"
		}

		client.Publish("homeassistant/sensor/BIG-DISK-ENERGY/PC_Monitor/state", 0, false, value)
		client.Publish("homeassistant/sensor/BIG-DISK-ENERGY/availability", 0, false, "online")
		time.Sleep(5 * time.Second)
	}

	if err != nil {
		fmt.Println(err)
	}
}

func isInactive() (bool, error) {
	conn, err := xgb.NewConn()
	if err != nil {
		return false, err
	}
	defer conn.Close()

	info := xproto.Setup(conn)
	screen := info.DefaultScreen(conn)

	if err := screensaver.Init(conn); err != nil {
		return false, err
	}

	rep, err := screensaver.QueryInfo(conn, xproto.Drawable(screen.Root)).Reply()
	if err != nil {
		return false, err
	}

	return rep.State == 1, nil
}
