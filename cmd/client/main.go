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

	fmt.Println("Starting Peril client...")

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

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		fmt.Println("Failed to get welcome message:", err)
		return
	}
	queueName := fmt.Sprintf("%s.%s", routing.PauseKey, username)

	_, _, err = pubsub.DeclareAndBind(amqpConn, routing.ExchangePerilDirect, queueName, routing.PauseKey, pubsub.Transient)
	if err != nil {
		fmt.Println("Failed to declare and bind queue:", err)
		return
	}

	gameState := gamelogic.NewGameState(username)
	err = pubsub.SubscribeJSON(amqpConn, routing.ExchangePerilDirect, queueName, routing.PauseKey, pubsub.Transient, HandlerPause(gameState))
	if err != nil {
		fmt.Println("Failed to subscribe:", err)
		return
	}

	err = pubsub.SubscribeJSON(
		amqpConn,
		string(routing.ExchangePerilTopic),
		fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, username),
		fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, "*"),
		pubsub.Transient,
		HandlerMove(gameState, amqpChan))
	if err != nil {
		fmt.Println("Failed to subscribe:", err)
		return
	}

	err = pubsub.SubscribeJSON(
		amqpConn,
		string(routing.ExchangePerilTopic),
		"war",
		fmt.Sprintf("%s.%s", routing.WarRecognitionsPrefix, "*"),
		pubsub.Durable,
		HandlerWar(gameState))
	if err != nil {
		fmt.Println("Failed to subscribe:", err)
		return
	}

	for {
		userInput := gamelogic.GetInput()
		if len(userInput) == 0 {
			continue
		}

		command := userInput[0]
		if command == "spawn" {
			err := gameState.CommandSpawn(userInput)
			if err != nil {
				fmt.Println("Failed to spawn:", err)
				continue
			}
		} else if command == "move" {
			move, err := gameState.CommandMove(userInput)
			if err != nil {
				fmt.Println("Failed to move:", err)
				continue
			}

			err = pubsub.PublishJSON(
				amqpChan,
				string(routing.ExchangePerilTopic),
				fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, username),
				move)
			if err != nil {
				fmt.Println("Failed to publish move:", err)
				break
			}
			fmt.Println("Move published correctly")

			fmt.Printf("Moved to %s successfully\n", move.ToLocation)
		} else if command == "status" {
			gameState.CommandStatus()
		} else if command == "help" {
			gamelogic.PrintClientHelp()
		} else if command == "spam" {
			fmt.Println("Spamming not allowed yet...")
		} else if command == "quit" {
			gamelogic.PrintQuit()
			break
		} else {
			fmt.Println("Unknown command: ", command)
			continue
		}
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
}
