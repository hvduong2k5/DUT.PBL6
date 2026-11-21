#!/bin/sh
# Run once by the Kafka operator, not by an application instance.
set -eu
: "${KAFKA_BOOTSTRAP_SERVERS:?Set Kafka bootstrap servers}"
: "${KAFKA_REPLICATION_FACTOR:?Set replication factor for this cluster}"
# kafka-topics.sh / kafka-configs.sh must be on PATH (Apache Kafka distribution).
kafka-topics.sh --bootstrap-server "$KAFKA_BOOTSTRAP_SERVERS" --create --if-not-exists --topic procurement.events.v1 --partitions 3 --replication-factor "$KAFKA_REPLICATION_FACTOR"
kafka-topics.sh --bootstrap-server "$KAFKA_BOOTSTRAP_SERVERS" --create --if-not-exists --topic procurement.events.dlq.v1 --partitions 3 --replication-factor "$KAFKA_REPLICATION_FACTOR"
kafka-configs.sh --bootstrap-server "$KAFKA_BOOTSTRAP_SERVERS" --alter --entity-type topics --entity-name procurement.events.dlq.v1 --add-config retention.ms=7776000000
