import { Module } from '@nestjs/common';
import { KafkaModule } from '../kafka/kafka.module';
import { StorageModule } from '../storage/storage.module';
import { ImageOptimizerService } from './image-optimizer.service';

@Module({
  imports: [StorageModule, KafkaModule],
  providers: [ImageOptimizerService],
  exports: [ImageOptimizerService],
})
export class OptimizerModule {}
