import { Injectable, Logger, OnModuleDestroy, OnModuleInit } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { Kafka, Producer } from 'kafkajs';

@Injectable()
export class KafkaService implements OnModuleInit, OnModuleDestroy {
  private readonly logger = new Logger(KafkaService.name);
  private readonly enabled: boolean;
  private readonly topic: string;
  private producer: Producer | null = null;

  constructor(private readonly configService: ConfigService) {
    this.enabled = this.configService.get<boolean>('kafka.enabled') ?? false;
    this.topic = this.configService.get<string>('kafka.topic') || 'media.events.v1';

    if (this.enabled) {
      const kafka = new Kafka({
        clientId: this.configService.get<string>('kafka.clientId') || 'media-service',
        brokers: this.configService.get<string[]>('kafka.brokers') || ['localhost:9092'],
      });
      this.producer = kafka.producer();
    }
  }

  async onModuleInit(): Promise<void> {
    if (!this.enabled || !this.producer) {
      return;
    }
    try {
      await this.producer.connect();
      this.logger.log(`Kafka producer connected (topic=${this.topic})`);
    } catch (error) {
      this.logger.warn(`Kafka connection failed: ${error?.message}`);
    }
  }

  async onModuleDestroy(): Promise<void> {
    if (this.producer) {
      await this.producer.disconnect().catch(() => undefined);
    }
  }

  async emitEvent(eventType: string, key: string, data: Record<string, unknown>): Promise<void> {
    const envelope = {
      specversion: '1.0',
      id: crypto.randomUUID(),
      source: 'https://omama.vn/services/media-service',
      type: eventType,
      time: new Date().toISOString(),
      data,
    };

    if (!this.enabled || !this.producer) {
      this.logger.debug(`[Kafka disabled] Event ${eventType}: ${JSON.stringify(data)}`);
      return;
    }

    await this.producer.send({
      topic: this.topic,
      messages: [
        {
          key,
          value: JSON.stringify(envelope),
        },
      ],
    });
  }
}
