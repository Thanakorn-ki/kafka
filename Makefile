.PHONY: all deps test build benchmark coveralls

up:
	@echo "Running app's service with kafka"
	docker-compose up

down:
	@echo "Goodbye~"
	docker-compose down

producer:
	@echo "exec producer command line"
	docker exec -it kafka sh  -c "/opt/kafka/bin/kafka-console-producer.sh --broker-list localhost:9092 --topic retain-offset"