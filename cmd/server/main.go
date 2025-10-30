package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {

	connString := "amqp://guest:guest@localhost:5672/"
	amqpConn, err := amqp.Dial(connString)
	if err != nil {
		fmt.Println("Failed to connect to RabbitMQ:", err)
		return
	}
	defer amqpConn.Close()

	amqpChan, err := amqpConn.Channel()
	if err != nil {
		fmt.Println("Failed to create RabbitMQ channel:", err)
		return
	}
	defer amqpChan.Close()

	_, _, err = pubsub.DeclareAndBind(amqpConn, routing.ExchangePerilTopic, routing.GameLogSlug, "game_logs.*", pubsub.Durable)
	if err != nil {
		fmt.Println("Failed to declare and bind RabbitMQ exchange and queue:", err)
		return
	}

	fmt.Println("Connected to RabbitMQ successfully")
	fmt.Println("Starting Peril server...")

	gamelogic.PrintServerHelp()

	for {
		userInput := gamelogic.GetInput()
		if len(userInput) == 0 {
			continue
		}

		command := userInput[0]

		if command == "pause" {
			fmt.Println("Pausing game...")
			pubsub.PublishJSON(amqpChan, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{
				IsPaused: true,
			})
		} else if command == "resume" {
			fmt.Println("Resuming game...")
			pubsub.PublishJSON(amqpChan, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{
				IsPaused: false,
			})
		} else if command == "quit" {
			fmt.Println("Quitting from the game...")
			break
		} else {
			fmt.Println("Unknown command: ", command)
		}
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan

	fmt.Println("\nShutting down Peril server...")
}
