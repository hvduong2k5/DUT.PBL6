import { ValidationPipe } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { NestFactory } from '@nestjs/core';
import { MicroserviceOptions, Transport } from '@nestjs/microservices';
import * as path from 'path';
import { AppModule } from './app.module';

async function bootstrap() {
  const app = await NestFactory.create(AppModule);
  const config = app.get(ConfigService);

  app.useGlobalPipes(
    new ValidationPipe({
      whitelist: true,
      transform: true,
      forbidNonWhitelisted: true,
    }),
  );

  const protoRoot =
    process.env.PROTO_ROOT || path.resolve(__dirname, '../../../packages/proto');
  const protoPath = path.join(protoRoot, 'media/v1/media_service.proto');
  const grpcPort = config.get<number>('grpcPort') || 9007;

  app.connectMicroservice<MicroserviceOptions>({
    transport: Transport.GRPC,
    options: {
      package: 'omamx.media.v1',
      protoPath,
      url: `0.0.0.0:${grpcPort}`,
    },
  });

  await app.startAllMicroservices();

  const httpPort = config.get<number>('httpPort') || 8007;
  await app.listen(httpPort);
}

bootstrap();
