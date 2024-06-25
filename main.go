package main

import (
	"encoding/json"
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
		MQTT_BROKER = "mqtt://localhost:1883"
		MQTT_USER   = os.Getenv("MQTT_USER")
		MQTT_PASS   = os.Getenv("MQTT_PASS")
		MQTT_CLIENT = "Linux-PC-Monitor"
	)

	if os.Getenv("MQTT_BROKER") != "" {
		MQTT_BROKER = os.Getenv("MQTT_BROKER")
	}

	if os.Getenv("MQTT_CLIENT") != "" {
		MQTT_CLIENT = os.Getenv("MQTT_CLIENT")
	}

	opts := mqtt.NewClientOptions().AddBroker(MQTT_BROKER)
	opts.SetClientID(MQTT_CLIENT)
	opts.SetUsername(MQTT_USER)
	opts.SetPassword(MQTT_PASS)

	client := mqtt.NewClient(opts)

	// TODO: Replace sleep with a loop and a timeout
	time.Sleep(10 * time.Second)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Println("Failed to connect to MQTT broker")
		panic(token.Error())
	} else {
		fmt.Println("Connected to MQTT broker")
		fmt.Println("Publishing integration message")
		configData, err := json.Marshal(configMsg())
		if err != nil {
			fmt.Println("Error while marshalling integration message")
			panic(err)
		}
		token = client.Publish("homeassistant/sensor/BIG-DISK-ENERGY/PC_Monitor/config", 0, true, configData)
		token.Wait()
		fmt.Println(token.Error())
		fmt.Println("Config message published")
	}

	// publish monitor availability
	continueSignal := make(chan bool, 1)
	// Giving initial value to start the cycle
	continueSignal <- true

	signalChannel := make(chan os.Signal, 2)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGALRM)
	go handleSignal(client, signalChannel, continueSignal)
	var err error
	var isIdle bool
	var value string
	for err == nil {
		// wait for a signal to continue the loop
		fmt.Println("Waiting for signal...")
		<-continueSignal
		isIdle, err = isInactive()

		if !isIdle {
			value = "PowerOn"
		} else {
			value = "PowerOff"
		}

		fmt.Println("Publishing state...")
		client.Publish("homeassistant/sensor/BIG-DISK-ENERGY/PC_Monitor/state", 0, false, value)
		client.Publish("homeassistant/sensor/BIG-DISK-ENERGY/availability", 0, false, "online")
		time.Sleep(5 * time.Second)
		// produce a signal to continue the loop
		continueSignal <- true
	}

	fmt.Println("Error while monitoring")
	fmt.Println(err)
	fmt.Println("Exiting...")
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

func handleSignal(client mqtt.Client, signalChannel chan os.Signal, continueSignal chan bool) {
	for signal := range signalChannel {
		switch signal {
		case syscall.SIGINT:
			// consume the signal to stop the loop
			<-continueSignal
			fmt.Println("Received an interrupt, cleaning...")
			client.Publish("homeassistant/sensor/BIG-DISK-ENERGY/PC_Monitor/state", 0, false, "PowerOff")
			client.Publish("homeassistant/sensor/BIG-DISK-ENERGY/availability", 0, false, "offline")
			client.Disconnect(250)
		case syscall.SIGALRM:
			fmt.Println("Got SIGALRM...")
			fmt.Println("Restarting monitoring...")
			// produce a signal to continue the loop
			continueSignal <- true
		}
	}
}

func configMsg() interface{} {
	type device struct {
		Identifiers  string `json:"identifiers"`
		Manufacturer string `json:"manufacturer"`
		Model        string `json:"model"`
		Name         string `json:"name"`
		Sw_version   string `json:"sw_version"`
	}

	type config struct {
		Availability_topic string `json:"availability_topic"`
		Icon               string `json:"icon"`
		Unique_id          string `json:"unique_id"`
		Device             device `json:"device"`
		Name               string `json:"name"`
		State_topic        string `json:"state_topic"`
	}

	return config{
		Availability_topic: "homeassistant/sensor/BIG-DISK-ENERGY/availability",
		Icon:               "mdi:monitor",
		Unique_id:          "b66cedc1-3c5a-4b79-b8d8-93a329bf5606",
		Device: device{
			Identifiers:  "hass.agent-BIG-DISK-ENERGY",
			Manufacturer: "LAB02 Research",
			Model:        "Microsoft Windows NT 10.0.22631.0",
			Name:         "BIG-DISK-ENERGY",
			Sw_version:   "2022.14.0",
		},
		Name:        "PC_Monitor",
		State_topic: "homeassistant/sensor/BIG-DISK-ENERGY/PC_Monitor/state",
	}
}
