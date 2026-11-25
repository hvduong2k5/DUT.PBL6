import { Injectable, Logger, OnModuleInit } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import {
  CopyObjectCommand,
  CreateBucketCommand,
  DeleteObjectCommand,
  GetObjectCommand,
  HeadBucketCommand,
  HeadObjectCommand,
  PutObjectCommand,
  S3Client,
} from '@aws-sdk/client-s3';
import { getSignedUrl } from '@aws-sdk/s3-request-presigner';

@Injectable()
export class StorageService implements OnModuleInit {
  private readonly logger = new Logger(StorageService.name);
  private readonly s3Client: S3Client;
  private readonly presignClient: S3Client;

  constructor(private readonly configService: ConfigService) {
    const endpoint = this.configService.get<string>('minio.endpoint');
    const publicEndpoint = this.configService.get<string>('minio.publicEndpoint');
    const region = this.configService.get<string>('minio.region') || 'us-east-1';
    const credentials = {
      accessKeyId: this.configService.get<string>('minio.accessKey') || 'minioadmin',
      secretAccessKey: this.configService.get<string>('minio.secretKey') || 'minioadmin',
    };

    this.s3Client = new S3Client({
      endpoint,
      region,
      credentials,
      forcePathStyle: true,
    });

    this.presignClient = new S3Client({
      endpoint: publicEndpoint || endpoint,
      region,
      credentials,
      forcePathStyle: true,
    });
  }

  async onModuleInit(): Promise<void> {
    if (process.env.SKIP_S3_INIT === 'true') {
      return;
    }
    try {
      await this.ensureBuckets();
    } catch (error) {
      this.logger.warn(`MinIO bucket initialization skipped or failed: ${error?.message}`);
    }
  }

  async ensureBuckets(): Promise<void> {
    const buckets = Object.values(
      this.configService.get<Record<string, string>>('minio.buckets') || {},
    );
    for (const bucket of buckets) {
      try {
        await this.s3Client.send(new HeadBucketCommand({ Bucket: bucket }));
      } catch {
        await this.s3Client.send(new CreateBucketCommand({ Bucket: bucket }));
        this.logger.log(`Created MinIO bucket: ${bucket}`);
      }
    }
  }

  async getPresignedPutUrl(
    bucket: string,
    key: string,
    contentType: string,
    expiresInSeconds = 900,
  ): Promise<string> {
    const command = new PutObjectCommand({
      Bucket: bucket,
      Key: key,
      ContentType: contentType,
    });
    return getSignedUrl(this.presignClient, command, { expiresIn: expiresInSeconds });
  }

  async getPresignedGetUrl(
    bucket: string,
    key: string,
    disposition: 'inline' | 'attachment' = 'inline',
    fileName?: string,
    expiresInSeconds = 900,
  ): Promise<string> {
    const safeName = (fileName || 'file').replace(/["\r\n]/g, '_');
    const responseContentDisposition =
      disposition === 'attachment' ? `attachment; filename="${safeName}"` : 'inline';

    const command = new GetObjectCommand({
      Bucket: bucket,
      Key: key,
      ResponseContentDisposition: responseContentDisposition,
    });
    return getSignedUrl(this.presignClient, command, { expiresIn: expiresInSeconds });
  }

  async headObject(
    bucket: string,
    key: string,
  ): Promise<{ contentLength: number; contentType?: string } | null> {
    try {
      const result = await this.s3Client.send(
        new HeadObjectCommand({ Bucket: bucket, Key: key }),
      );
      return {
        contentLength: result.ContentLength ?? 0,
        contentType: result.ContentType,
      };
    } catch {
      return null;
    }
  }

  async getObjectBuffer(bucket: string, key: string): Promise<Buffer> {
    const result = await this.s3Client.send(
      new GetObjectCommand({ Bucket: bucket, Key: key }),
    );
    const bytes = await result.Body?.transformToByteArray();
    if (!bytes) {
      throw new Error(`Empty object body for ${bucket}/${key}`);
    }
    return Buffer.from(bytes);
  }

  async putObjectBuffer(
    bucket: string,
    key: string,
    body: Buffer,
    contentType: string,
  ): Promise<void> {
    await this.s3Client.send(
      new PutObjectCommand({
        Bucket: bucket,
        Key: key,
        Body: body,
        ContentType: contentType,
      }),
    );
  }

  async copyObject(
    sourceBucket: string,
    sourceKey: string,
    destBucket: string,
    destKey: string,
  ): Promise<void> {
    await this.s3Client.send(
      new CopyObjectCommand({
        Bucket: destBucket,
        Key: destKey,
        CopySource: `${sourceBucket}/${sourceKey}`,
      }),
    );
  }

  async deleteObject(bucket: string, key: string): Promise<void> {
    await this.s3Client.send(
      new DeleteObjectCommand({
        Bucket: bucket,
        Key: key,
      }),
    );
  }
}
