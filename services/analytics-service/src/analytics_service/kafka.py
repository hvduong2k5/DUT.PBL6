import time
from uuid import uuid4

from confluent_kafka import Consumer, KafkaException, TopicCollection, TopicPartition
from confluent_kafka.admin import AdminClient

from . import db
from .config import TOPIC
from .events import InvalidRecord, OrderAdapter, digest
from .metrics import LAG, RECORDS


def describe(settings):
    admin = AdminClient(settings.kafka())
    futures = admin.describe_topics(
        TopicCollection([TOPIC]),
        request_timeout=3,
    )
    topic = futures[TOPIC].result(timeout=4)
    return str(topic.topic_id), {part.id for part in topic.partitions}


def consumer(settings, group):
    return Consumer(
        {
            **settings.kafka(),
            "group.id": group,
            "enable.auto.commit": False,
            "enable.auto.offset.store": False,
            "auto.offset.reset": "error",
            "allow.auto.create.topics": False,
            "partition.assignment.strategy": "range",
        }
    )


def read_exact(client, partition, offset):
    client.assign([TopicPartition(TOPIC, partition, offset)])
    deadline = time.monotonic() + 3
    while time.monotonic() < deadline:
        message = client.poll(0.25)
        if message is not None:
            if message.error() or message.offset() != offset:
                break
            return message
    raise db.Conflict("Original Kafka record is unavailable")


def bootstrap(settings, identity, offsets):
    actual, partitions = describe(settings)
    if identity != actual or set(offsets) != partitions:
        raise db.Conflict("Bootstrap manifest does not match broker topic identity and partitions")
    with db.connect(settings) as conn:
        if db.cursors(conn, settings.pipeline):
            db.bootstrap(conn, settings.pipeline, identity, offsets)
            return
    client = consumer(settings, f"{settings.pipeline}-bootstrap")
    try:
        for partition, offset in offsets.items():
            low, high = client.get_watermark_offsets(TopicPartition(TOPIC, partition), timeout=3)
            if not low <= offset <= high:
                raise db.Conflict("Bootstrap offset is outside broker retention")
        with db.connect(settings) as conn:
            db.bootstrap(conn, settings.pipeline, identity, offsets)
    finally:
        client.close()


class Collector:
    def __init__(self, settings, stop):
        self.settings = settings
        self.pipeline = settings.pipeline
        self.stop = stop
        self.owner = str(uuid4())
        self.adapter = OrderAdapter(settings.schema_dir)
        self.pending = set()
        self.epochs = {}
        self.identity = ""

    def inspect_topic(self):
        identity, partitions = describe(self.settings)
        with db.connect(self.settings) as conn:
            rows = db.cursors(conn, self.pipeline)
            if not rows:
                return False
            if any(row["topic_identity"] != identity for row in rows):
                db.mark_gap(conn, self.pipeline, "TOPIC_ID_CHANGED")
                return False
            if partitions != {row["partition_id"] for row in rows}:
                db.mark_gap(conn, self.pipeline, "PARTITIONS_CHANGED")
                return False
        self.identity = identity
        return True

    def assigned(self, client, partitions):
        with db.connect(self.settings) as conn:
            offsets = {
                row["partition_id"]: row["next_offset"] for row in db.cursors(conn, self.pipeline)
            }
        client.assign(
            [
                TopicPartition(item.topic, item.partition, offsets[item.partition])
                for item in partitions
            ]
        )
        client.pause(partitions)
        self.pending.update(item.partition for item in partitions)

    def revoked(self, client, partitions):
        revoked = {item.partition for item in partitions}
        self.pending.difference_update(revoked)
        for partition in revoked:
            self.epochs.pop(partition, None)
        with db.connect(self.settings) as conn:
            db.release(conn, self.pipeline, self.owner, revoked)

    def start_partition(self, client, partition):
        with db.connect(self.settings) as conn:
            lease = db.claim(conn, self.pipeline, self.identity, partition, self.owner)
        if lease is None:
            return
        position = TopicPartition(TOPIC, partition, lease["next_offset"])
        low, high = client.get_watermark_offsets(position, timeout=2)
        if not low <= position.offset <= high:
            with db.connect(self.settings) as conn:
                db.mark_gap(conn, self.pipeline, "OFFSET_OUT_OF_RANGE", partition)
            self.pending.discard(partition)
            return
        anchor = lease["last_record_digest"]
        if anchor is not None and low < position.offset:
            probe = consumer(self.settings, f"{self.pipeline}-anchor")
            try:
                previous = read_exact(probe, partition, position.offset - 1)
                valid = digest(previous.value() or b"") == anchor
            except db.Conflict:
                valid = False
            finally:
                probe.close()
            if not valid:
                with db.connect(self.settings) as conn:
                    db.mark_gap(conn, self.pipeline, "REPLAY_ANCHOR_MISMATCH", partition)
                self.pending.discard(partition)
                return
        client.seek(position)
        client.resume([position])
        self.epochs[partition] = lease["lease_epoch"]
        self.pending.discard(partition)

    def process(self, client, message):
        partition = message.partition()
        if partition not in self.epochs:
            return
        raw = message.value() or b""
        event = None
        error = None
        try:
            event = self.adapter.decode(message.key(), raw)
        except InvalidRecord as exc:
            error = str(exc)
        with db.connect(self.settings) as conn:
            db.pipeline_lock(conn, self.pipeline)
            lease = db.claim(conn, self.pipeline, self.identity, partition, self.owner)
            if not lease or lease["lease_epoch"] != self.epochs[partition]:
                position = TopicPartition(TOPIC, partition)
                client.pause([position])
                self.pending.add(partition)
                self.epochs.pop(partition)
                return
            next_offset = db.ingest(
                conn,
                self.pipeline,
                self.identity,
                partition,
                message.offset(),
                self.owner,
                lease["lease_epoch"],
                raw,
                event,
                error,
            )
        if next_offset is None:
            client.pause([TopicPartition(TOPIC, partition)])
            self.epochs.pop(partition)
            return
        client.commit(offsets=[TopicPartition(TOPIC, partition, next_offset)], asynchronous=True)
        RECORDS.labels("invalid" if error else "accepted" if event else "ignored").inc()

    def run(self):
        if not self.inspect_topic():
            self.stop.wait(3)
            return
        client = consumer(self.settings, self.pipeline)
        try:
            client.subscribe(
                [TOPIC], on_assign=self.assigned, on_revoke=self.revoked, on_lost=self.revoked
            )
            next_inspection = time.monotonic() + 5
            while not self.stop.is_set():
                message = client.poll(0.25)
                if message is not None:
                    if message.error():
                        raise KafkaException(message.error())
                    self.process(client, message)
                for partition in list(self.pending):
                    self.start_partition(client, partition)
                if time.monotonic() >= next_inspection:
                    if not self.inspect_topic():
                        return
                    for partition, epoch in list(self.epochs.items()):
                        position = TopicPartition(TOPIC, partition)
                        low, high = client.get_watermark_offsets(position, timeout=2)
                        with db.connect(self.settings) as conn:
                            db.pipeline_lock(conn, self.pipeline)
                            lease = db.claim(
                                conn, self.pipeline, self.identity, partition, self.owner
                            )
                            if not lease or lease["lease_epoch"] != epoch:
                                client.pause([position])
                                self.pending.add(partition)
                                self.epochs.pop(partition)
                                continue
                            if not low <= lease["next_offset"] <= high:
                                db.mark_gap(conn, self.pipeline, "OFFSET_OUT_OF_RANGE", partition)
                                client.pause([position])
                                self.epochs.pop(partition)
                            LAG.labels(str(partition)).set(max(0, high - lease["next_offset"]))
                    next_inspection = time.monotonic() + 5
        finally:
            client.close()
