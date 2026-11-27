import { Module } from '@nestjs/common';
import { ConfigModule } from '@nestjs/config';
import configuration from './config/configuration';
import { MediaGrpcModule } from './modules/grpc/media-grpc.module';
import { KafkaModule } from './modules/kafka/kafka.module';
import { MediaModule } from './modules/media/media.module';
import { OptimizerModule } from './modules/optimizer/optimizer.module';
import { StorageModule } from './modules/storage/storage.module';
import { PrismaModule } from './prisma/prisma.module';

@Module({
  imports: [
    ConfigModule.forRoot({
      isGlobal: true,
      load: [configuration],
    }),
    PrismaModule,
    StorageModule,
    KafkaModule,
    OptimizerModule,
    MediaModule,
    MediaGrpcModule,
  ],
})
export class AppModule {}
