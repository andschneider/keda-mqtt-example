package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// config sets the client configuration, ensuring client ids are unique
// to prevent the broker forcing disconnects.
func config(host string, publish bool) mqtt.Client {
	var cid string
	if publish {
		cid = fmt.Sprintf("%s-%d", "pub", os.Getpid())
	} else {
		cid = fmt.Sprintf("%s-%d", "sub", os.Getpid())
	}

	// configure client
	opts := mqtt.NewClientOptions()
	opts.AddBroker(host)
	opts.SetClientID(cid)
	opts.SetCleanSession(true)
	opts.KeepAlive = 30
	opts.PingTimeout = 15 * time.Second

	return mqtt.NewClient(opts)
}

func main() {
	publish := flag.Bool("pub", false, "Pass in to publish a message, otherwise it will subscribe.")
	count := flag.Int("count", 1, "Number of messages to send (with a 1 second delay in between)")
	retain := flag.Bool("retain", false, "Pass in to send a publish message with the retained flag (need to pass in -pub too)")
	clear := flag.Bool("clear", false, "Pass in to clear any retained messages in a topic (need to pass in -pub too)")
	host := flag.String("host", "tcp://test.mosquitto.org:1883", "MQTT broker to use.")
	topic := flag.String("topic", "keda-example", "Topic to interact with")
	flag.Parse()

	c := config(*host, *publish)
	qos := 0 // TODO test with higher QoS

	// attempt connection to broker
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		fmt.Printf("could not connect to broker: %v\n", token.Error())
		os.Exit(1)
	}
	defer c.Disconnect(1000) // wait 1000ms before disconnecting

	switch {
	case *clear:
		// Send a retained message with a blank message body to clear any retained messages
		token := c.Publish(*topic, byte(qos), true, "")
		token.Wait()
		fmt.Printf("sent clear retained command to topic: %v\n", *topic)
	case *publish && *retain:
		// Send a retained message
		message := "status: up"
		token := c.Publish(*topic, byte(qos), *retain, message)
		token.Wait()
		fmt.Printf("sent retained message: %q, to topic: %v\n", message, *topic)
	case *publish:
		// Send at least 1 normal message
		for i := 1; i <= *count; i++ {
			message := fmt.Sprintf("sensor: %d", i)
			token := c.Publish(*topic, byte(qos), false, message)
			token.Wait()
			fmt.Printf("sent message: %q, to topic: %v\n", message, *topic)
			time.Sleep(1 * time.Second)
		}
	default:
		// Subscribe to a topic until a shutdown signal
		retained := false
		if token := c.Subscribe(*topic, byte(qos), func(client mqtt.Client, message mqtt.Message) {
			fmt.Printf("received message: %s, retained: %t\n", message.Payload(), message.Retained())
			// Check if the message is retained. This is only true for a message received on first connection. If
			// a message that has been marked as retained is received after connection, it will still be marked
			// as false.
			if message.Retained() {
				retained = true
			}
		}); token.Wait() && token.Error() != nil {
			fmt.Println(token.Error())
			os.Exit(1)
		}
		fmt.Printf("subscribed to topic: %v\n", *topic)

		// process shutdown signal
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt)
		signal.Notify(sig, syscall.SIGTERM)
		<-sig
	}
}
