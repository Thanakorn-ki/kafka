package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Shopify/sarama"
	"github.com/sirupsen/logrus"
)

var (
	version   = "1.1.0"
	keepAlive = 5 * time.Second
)

var log  *logrus.Logger

func init () {
	log = logrus.New()
	log.SetFormatter(&logrus.TextFormatter{})
}

func main () {
	ver, err := sarama.ParseKafkaVersion(version)
	if err != nil {
		log.Panicf("Error parsing Kafka version: %v", err)
	}

	config := sarama.NewConfig()
	config.Version = ver
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.Net.KeepAlive = keepAlive
	config.Consumer.Offsets.Retention = 10 * time.Second
	// config.Consumer.Offsets.AutoCommit.Enable = true
	// config.Consumer.Offsets.AutoCommit.Interval = 1 * time.Second
	/**
	 * Setup a new Sarama consumer group
	 */

	 
	 consumer := Consumer{
		ready: make(chan bool),
	}

	ctx, cancel := context.WithCancel(context.Background())
	client, err := sarama.NewConsumerGroup(strings.Split(os.Getenv("KAFKA_BROKERS"), ","), os.Getenv("KAFKA_CONSUMER_GROUP"), config)
	if err != nil {
		log.Panicf("Error creating consumer group client: %v", err)
	}
	
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			// `Consume` should be called inside an infinite loop, when a
			// server-side rebalance happens, the consumer session will need to be
			// recreated to get the new claims
			if err := client.Consume(ctx, strings.Split(os.Getenv("KAFKA_TOPIC"), ","), &consumer); err != nil {
				log.Panicf("Error from consumer: %v", err)
			}
			// check if context was cancelled, signaling that the consumer should stop
			if ctx.Err() != nil {
				return
			}
			consumer.ready = make(chan bool)
		}
	}()


	<-consumer.ready // Await till the consumer has been set up
	log.Print("Sarama consumer up and running!...")

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-ctx.Done():
		log.Println("terminating: context cancelled")
	case <-sigterm:
		log.Println("terminating: via signal")
	}
	cancel()
	wg.Wait()
	if err = client.Close(); err != nil {
		log.Panicf("Error closing client: %v", err)
	}
}

// Consumer represents a Sarama consumer group consumer
type Consumer struct {

	ready chan bool
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (consumer *Consumer) Setup(sarama.ConsumerGroupSession) error {
	// Mark the consumer as ready
	close(consumer.ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (consumer *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (consumer *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	log.Infof("session %v", session.Claims())
	log.Infof("initial offset %d", claim.InitialOffset())
	// 2020-03-04 09:37:30.124 +0000 UTC, topic = retain-offset
	// NOTE:
	// Do not move the code below to a goroutine.
	// The `ConsumeClaim` itself is called within a goroutine, see:
	// https://github.com/Shopify/sarama/blob/master/consumer_group.go#L27-L29
	for message := range claim.Messages() {
		log.Infof("offset %d", message.Offset)
		log.Infof("partition %d", message.Partition)
		log.Infof("header %+v", message.Headers)
		log.Infof("Message claimed: value = %s, timestamp = %v, topic = %s", string(message.Value), message.Timestamp, message.Topic)
		session.MarkMessage(message, "")
	}

	return nil
}




