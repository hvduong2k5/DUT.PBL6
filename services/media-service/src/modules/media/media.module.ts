import { Module } from '@nestjs/common';
import { KafkaModule } from '../kafka/kafka.module';
import { OptimizerModule } from '../optimizer/optimizer.module';
import { StorageModule } from '../storage/storage.module';
import { MediaController } from './media.controller';
import { MediaService } from './media.service';

@Module({
  imports: [StorageModule, KafkaModule, OptimizerModule],
  controllers: [MediaController],
  providers: [MediaService],
  exports: [MediaService],
})
export class MediaModule {}
