import {
  BadRequestException,
  Injectable,
  NotFoundException,
} from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { AccessLevel, MediaFile, MediaPurpose, MediaStatus } from '@prisma/client';
import * as crypto from 'crypto';
import * as path from 'path';
import 'multer';
import { PrismaService } from '../../prisma/prisma.service';
import { KafkaService } from '../kafka/kafka.service';
import { ImageOptimizerService } from '../optimizer/image-optimizer.service';
import { StorageService } from '../storage/storage.service';
import { CreateUploadSessionDto } from './dto/create-session.dto';

const ALLOWED_MIME_TYPES_BY_PURPOSE: Record<MediaPurpose, Set<string>> = {
  CATALOG_PRODUCT: new Set(['image/jpeg', 'image/png', 'image/webp']),
  CATALOG_CATEGORY: new Set(['image/jpeg', 'image/png', 'image/webp']),
  AVATAR: new Set(['image/jpeg', 'image/png', 'image/webp']),
  OCOP_CERTIFICATE: new Set(['application/pdf', 'image/jpeg', 'image/png', 'image/webp']),
  SHIPPING_VIDEO: new Set(['video/mp4', 'video/quicktime', 'video/webm']),
  B2B_DOCUMENT: new Set(['application/pdf', 'image/jpeg', 'image/png']),
  REFUND_EVIDENCE: new Set(['image/jpeg', 'image/png', 'image/webp', 'video/mp4']),
};

const ENTITY_PREFIX_BY_PURPOSE: Record<MediaPurpose, string> = {
  CATALOG_PRODUCT: 'products',
  CATALOG_CATEGORY: 'categories',
  AVATAR: 'avatars',
  OCOP_CERTIFICATE: 'certificates',
  SHIPPING_VIDEO: 'videos',
  B2B_DOCUMENT: 'b2b-licenses',
  REFUND_EVIDENCE: 'refunds',
};

@Injectable()
export class MediaService {
  constructor(
    private readonly prisma: PrismaService,
    private readonly storage: StorageService,
    private readonly kafka: KafkaService,
    private readonly optimizer: ImageOptimizerService,
    private readonly config: ConfigService,
  ) {}

  private resolveTargetBucket(purpose: MediaPurpose): {
    bucket: string;
    accessLevel: AccessLevel;
  } {
    switch (purpose) {
      case MediaPurpose.CATALOG_PRODUCT:
      case MediaPurpose.CATALOG_CATEGORY:
        // Ảnh chưa được Catalog Moderator duyệt phải giữ trong vùng private
        return {
          bucket: this.config.get<string>('minio.buckets.private') || 'om-private',
          accessLevel: AccessLevel.PRIVATE_RESTRICTED,
        };
      case MediaPurpose.AVATAR:
        return {
          bucket: this.config.get<string>('minio.buckets.catalog') || 'om-catalog',
          accessLevel: AccessLevel.PUBLIC_READ,
        };
      case MediaPurpose.OCOP_CERTIFICATE:
        return {
          bucket: this.config.get<string>('minio.buckets.traceability') || 'om-traceability',
          accessLevel: AccessLevel.PUBLIC_READ,
        };
      case MediaPurpose.SHIPPING_VIDEO:
        return {
          bucket: this.config.get<string>('minio.buckets.fulfillment') || 'om-fulfillment',
          accessLevel: AccessLevel.PRIVATE_RESTRICTED,
        };
      case MediaPurpose.B2B_DOCUMENT:
      case MediaPurpose.REFUND_EVIDENCE:
      default:
        return {
          bucket: this.config.get<string>('minio.buckets.documents') || 'om-documents',
          accessLevel: AccessLevel.PRIVATE_RESTRICTED,
        };
    }
  }

  /**
   * Chuẩn hóa objectKey theo quy tắc: {entity-type}/{YYYY}/{MM}/{DD}/{uuid}.{ext}
   */
  generateObjectKey(purpose: MediaPurpose, fileName: string, fileId: string): string {
    const now = new Date();
    const yyyy = String(now.getUTCFullYear());
    const mm = String(now.getUTCMonth() + 1).padStart(2, '0');
    const dd = String(now.getUTCDate()).padStart(2, '0');
    const entityType = ENTITY_PREFIX_BY_PURPOSE[purpose] || 'misc';

    const rawExt = path.extname(fileName || '').toLowerCase().replace(/[^a-z0-9.]/g, '');
    const ext = rawExt && rawExt.length <= 10 ? rawExt : '.bin';

    return `${entityType}/${yyyy}/${mm}/${dd}/${fileId}${ext}`;
  }

  async createUploadSession(dto: CreateUploadSessionDto, uploaderId: string) {
    const allowedMimes = ALLOWED_MIME_TYPES_BY_PURPOSE[dto.purpose];
    if (!allowedMimes || !allowedMimes.has(dto.mediaType.toLowerCase())) {
      throw new BadRequestException(
        `Unsupported mediaType '${dto.mediaType}' for purpose '${dto.purpose}'`,
      );
    }

    const fileId = crypto.randomUUID();
    const sessionId = `sess_${crypto.randomBytes(12).toString('hex')}`;
    const objectKey = this.generateObjectKey(dto.purpose, dto.fileName, fileId);
    const { bucket, accessLevel } = this.resolveTargetBucket(dto.purpose);
    const expirySeconds = this.config.get<number>('presignedPutExpirySeconds') || 900;
    const expiresAt = new Date(Date.now() + expirySeconds * 1000);

    const uploadUrl = await this.storage.getPresignedPutUrl(
      bucket,
      objectKey,
      dto.mediaType,
      expirySeconds,
    );

    await this.prisma.$transaction([
      this.prisma.mediaFile.create({
        data: {
          id: fileId,
          objectKey,
          bucketName: bucket,
          fileName: dto.fileName,
          mediaType: dto.mediaType.toLowerCase(),
          sizeBytes: BigInt(dto.sizeBytes),
          sha256Checksum: dto.sha256 || null,
          purpose: dto.purpose,
          accessLevel,
          status: MediaStatus.AWAITING_UPLOAD,
          uploaderId,
        },
      }),
      this.prisma.uploadSession.create({
        data: {
          id: sessionId,
          fileId,
          uploadUrl,
          expiresAt,
          status: 'ACTIVE',
        },
      }),
    ]);

    return {
      uploadSessionId: sessionId,
      fileId,
      objectKey,
      bucket,
      uploadUrl,
      expiresAt: expiresAt.toISOString(),
      requiredHeaders: {
        'Content-Type': dto.mediaType,
      },
    };
  }

  async completeUploadSession(sessionId: string) {
    const session = await this.prisma.uploadSession.findUnique({
      where: { id: sessionId },
      include: { file: true },
    });
    if (!session) {
      throw new NotFoundException(`Upload session '${sessionId}' not found`);
    }
    if (session.status === 'COMPLETED') {
      return this.serializeMedia(session.file);
    }

    const head = await this.storage.headObject(session.file.bucketName, session.file.objectKey);
    if (!head) {
      throw new BadRequestException(
        `Object '${session.file.objectKey}' has not been uploaded to storage yet`,
      );
    }

    if (head.contentLength <= 0 || head.contentLength !== Number(session.file.sizeBytes)) {
      throw new BadRequestException(
        `Uploaded size mismatch: expected ${session.file.sizeBytes} bytes, got ${head.contentLength} bytes`,
      );
    }

    const [updatedFile] = await this.prisma.$transaction([
      this.prisma.mediaFile.update({
        where: { id: session.fileId },
        data: { status: MediaStatus.PROCESSING },
      }),
      this.prisma.uploadSession.update({
        where: { id: sessionId },
        data: { status: 'COMPLETED' },
      }),
    ]);

    await this.kafka.emitEvent('vn.omama.media.file.uploaded.v1', updatedFile.objectKey, {
      fileId: updatedFile.id,
      objectKey: updatedFile.objectKey,
      bucket: updatedFile.bucketName,
      purpose: updatedFile.purpose,
      mediaType: updatedFile.mediaType,
      sizeBytes: Number(updatedFile.sizeBytes),
    });

    // Trigger async sharp worker (non-blocking HTTP response)
    setImmediate(() => {
      this.optimizer.processFile(updatedFile.id).catch(() => undefined);
    });

    return this.serializeMedia(updatedFile);
  }

  async getFile(fileId: string) {
    const file = await this.findByIdOrThrow(fileId);
    const expirySeconds = this.config.get<number>('presignedGetExpirySeconds') || 900;
    const viewUrl = await this.storage.getPresignedGetUrl(
      file.bucketName,
      file.objectKey,
      'inline',
      file.fileName,
      expirySeconds,
    );
    const downloadUrl = await this.storage.getPresignedGetUrl(
      file.bucketName,
      file.objectKey,
      'attachment',
      file.fileName,
      expirySeconds,
    );

    return {
      ...this.serializeMedia(file),
      viewUrl,
      downloadUrl,
    };
  }

  async getAccessUrl(fileId: string, disposition: 'inline' | 'attachment', expirySeconds = 900) {
    const file = await this.findByIdOrThrow(fileId);
    const ttl = expirySeconds > 0 ? Math.min(expirySeconds, 3600) : 900;
    const url = await this.storage.getPresignedGetUrl(
      file.bucketName,
      file.objectKey,
      disposition,
      file.fileName,
      ttl,
    );
    return {
      fileId: file.id,
      url,
      disposition,
      expiresAt: new Date(Date.now() + ttl * 1000).toISOString(),
    };
  }

  async directUpload(file: Express.Multer.File, purpose: MediaPurpose, uploaderId: string) {
    if (!file) {
      throw new BadRequestException('File is required');
    }
    const allowedMimes = ALLOWED_MIME_TYPES_BY_PURPOSE[purpose];
    if (!allowedMimes || !allowedMimes.has(file.mimetype.toLowerCase())) {
      throw new BadRequestException(`Unsupported mediaType '${file.mimetype}'`);
    }

    const fileId = crypto.randomUUID();
    const objectKey = this.generateObjectKey(purpose, file.originalname, fileId);
    const { bucket, accessLevel } = this.resolveTargetBucket(purpose);
    const sha256 = crypto.createHash('sha256').update(file.buffer).digest('hex');

    await this.storage.putObjectBuffer(bucket, objectKey, file.buffer, file.mimetype);

    const record = await this.prisma.mediaFile.create({
      data: {
        id: fileId,
        objectKey,
        bucketName: bucket,
        fileName: file.originalname,
        mediaType: file.mimetype.toLowerCase(),
        sizeBytes: BigInt(file.size),
        sha256Checksum: sha256,
        purpose,
        accessLevel,
        status: MediaStatus.PROCESSING,
        uploaderId,
      },
    });

    await this.optimizer.processFile(record.id);
    const updated = await this.findByIdOrThrow(record.id);
    return this.serializeMedia(updated);
  }

  async softDelete(fileId: string) {
    await this.findByIdOrThrow(fileId);
    const updated = await this.prisma.mediaFile.update({
      where: { id: fileId },
      data: { status: MediaStatus.ARCHIVED },
    });
    return this.serializeMedia(updated);
  }

  async validateReference(fileId?: string, objectKey?: string, expectedPurpose?: string) {
    const file = fileId
      ? await this.prisma.mediaFile.findUnique({ where: { id: fileId } })
      : objectKey
        ? await this.prisma.mediaFile.findUnique({ where: { objectKey } })
        : null;

    if (!file) {
      return {
        isValid: false,
        status: 'NOT_FOUND',
        objectKey: objectKey || '',
        mimeType: '',
        sizeBytes: 0,
        sha256Checksum: '',
      };
    }

    const purposeMatches = !expectedPurpose || file.purpose === expectedPurpose;
    const isValid = purposeMatches && file.status === MediaStatus.READY;

    return {
      isValid,
      status: file.status,
      objectKey: file.objectKey,
      mimeType: file.mediaType,
      sizeBytes: Number(file.sizeBytes),
      sha256Checksum: file.sha256Checksum || '',
    };
  }

  async bindToEntity(
    fileId: string | undefined,
    objectKey: string | undefined,
    entityType: string,
    entityId: string,
  ): Promise<boolean> {
    const file = fileId
      ? await this.prisma.mediaFile.findUnique({ where: { id: fileId } })
      : objectKey
        ? await this.prisma.mediaFile.findUnique({ where: { objectKey } })
        : null;
    if (!file) {
      return false;
    }

    await this.prisma.mediaFile.update({
      where: { id: file.id },
      data: {
        boundEntityType: entityType,
        boundEntityId: entityId,
      },
    });
    return true;
  }

  async publishMedia(
    fileId?: string,
    objectKey?: string,
  ): Promise<{ success: boolean; publicObjectKey: string }> {
    const file = fileId
      ? await this.prisma.mediaFile.findUnique({ where: { id: fileId } })
      : objectKey
        ? await this.prisma.mediaFile.findUnique({ where: { objectKey } })
        : null;
    if (!file) {
      return { success: false, publicObjectKey: '' };
    }

    const publicBucket = this.config.get<string>('minio.buckets.catalog') || 'om-catalog';
    if (file.bucketName !== publicBucket) {
      const keysToCopy = new Set<string>([file.objectKey]);
      const variants = (file.variantsMeta as Record<string, string>) || {};
      Object.values(variants).forEach((k) => {
        if (k) keysToCopy.add(k);
      });

      for (const key of keysToCopy) {
        await this.storage.copyObject(file.bucketName, key, publicBucket, key);
      }

      await this.prisma.mediaFile.update({
        where: { id: file.id },
        data: {
          bucketName: publicBucket,
          accessLevel: AccessLevel.PUBLIC_READ,
        },
      });
    }

    return { success: true, publicObjectKey: file.objectKey };
  }

  private async findByIdOrThrow(fileId: string): Promise<MediaFile> {
    const file = await this.prisma.mediaFile.findUnique({ where: { id: fileId } });
    if (!file) {
      throw new NotFoundException(`Media file '${fileId}' not found`);
    }
    return file;
  }

  private serializeMedia(file: MediaFile) {
    return {
      fileId: file.id,
      objectKey: file.objectKey,
      bucket: file.bucketName,
      fileName: file.fileName,
      mediaType: file.mediaType,
      sizeBytes: Number(file.sizeBytes),
      sha256Checksum: file.sha256Checksum,
      purpose: file.purpose,
      accessLevel: file.accessLevel,
      status: file.status,
      variantsMeta: file.variantsMeta,
      createdAt: file.createdAt.toISOString(),
      updatedAt: file.updatedAt.toISOString(),
    };
  }
}
