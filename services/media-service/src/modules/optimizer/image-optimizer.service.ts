import { Injectable, Logger } from '@nestjs/common';
import { MediaStatus } from '@prisma/client';
import sharp = require('sharp');
import { PrismaService } from '../../prisma/prisma.service';
import { KafkaService } from '../kafka/kafka.service';
import { StorageService } from '../storage/storage.service';

const SUPPORTED_IMAGE_TYPES = new Set([
  'image/jpeg',
  'image/png',
  'image/webp',
]);

const THUMBNAIL_SIZES = [150, 300, 600] as const;

@Injectable()
export class ImageOptimizerService {
  private readonly logger = new Logger(ImageOptimizerService.name);

  constructor(
    private readonly prisma: PrismaService,
    private readonly storage: StorageService,
    private readonly kafka: KafkaService,
  ) {}

  isSupportedImage(mediaType: string): boolean {
    return SUPPORTED_IMAGE_TYPES.has(mediaType.toLowerCase());
  }

  async processFile(fileId: string): Promise<void> {
    const media = await this.prisma.mediaFile.findUnique({ where: { id: fileId } });
    if (!media) {
      return;
    }

    // Non-image files (video, pdf) skip sharp optimization and become READY directly
    if (!this.isSupportedImage(media.mediaType)) {
      await this.prisma.mediaFile.update({
        where: { id: fileId },
        data: { status: MediaStatus.READY },
      });
      await this.kafka.emitEvent('vn.omama.media.file.derivatives_ready.v1', media.objectKey, {
        fileId: media.id,
        objectKey: media.objectKey,
        bucket: media.bucketName,
        variants: {},
      });
      return;
    }

    try {
      const inputBuffer = await this.storage.getObjectBuffer(media.bucketName, media.objectKey);
      const basePath = media.objectKey.replace(/\.[^/.]+$/, '');
      const webpKey = `${basePath}.webp`;

      const webpBuffer = await sharp(inputBuffer).webp({ quality: 85 }).toBuffer();
      await this.storage.putObjectBuffer(media.bucketName, webpKey, webpBuffer, 'image/webp');

      const variantsMeta: Record<string, string> = {
        original: media.objectKey,
        webp: webpKey,
      };

      for (const size of THUMBNAIL_SIZES) {
        const thumbKey = `${basePath}_${size}.webp`;
        const thumbBuffer = await sharp(inputBuffer)
          .resize(size, size, { fit: 'cover' })
          .webp({ quality: 82 })
          .toBuffer();
        await this.storage.putObjectBuffer(media.bucketName, thumbKey, thumbBuffer, 'image/webp');
        variantsMeta[`thumb_${size}`] = thumbKey;
      }

      await this.prisma.mediaFile.update({
        where: { id: fileId },
        data: {
          status: MediaStatus.READY,
          variantsMeta,
        },
      });

      await this.kafka.emitEvent('vn.omama.media.file.derivatives_ready.v1', media.objectKey, {
        fileId: media.id,
        objectKey: media.objectKey,
        bucket: media.bucketName,
        variants: variantsMeta,
      });

      this.logger.log(`Processed sharp WebP & thumbnails for fileId=${fileId}`);
    } catch (error) {
      const msg = error instanceof Error ? error.message : String(error);
      this.logger.error(`Sharp optimization failed for fileId=${fileId}: ${msg}`);
      await this.prisma.mediaFile.update({
        where: { id: fileId },
        data: { status: MediaStatus.PROCESSING_FAILED },
      });
    }
  }
}
