import { Module } from '@nestjs/common';
import { MediaModule } from '../media/media.module';
import { MediaGrpcController } from './media-grpc.controller';

@Module({
  imports: [MediaModule],
  controllers: [MediaGrpcController],
})
export class MediaGrpcModule {}
