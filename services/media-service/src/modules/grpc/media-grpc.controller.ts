import { Controller } from '@nestjs/common';
import { GrpcMethod } from '@nestjs/microservices';
import { MediaService } from '../media/media.service';

@Controller()
export class MediaGrpcController {
  constructor(private readonly mediaService: MediaService) {}

  @GrpcMethod('MediaInternalService', 'ValidateMediaReference')
  async validateMediaReference(data: {
    fileId?: string;
    objectKey?: string;
    expectedPurpose?: string;
  }) {
    return this.mediaService.validateReference(
      data.fileId,
      data.objectKey,
      data.expectedPurpose,
    );
  }

  @GrpcMethod('MediaInternalService', 'GetPresignedAccessUrl')
  async getPresignedAccessUrl(data: {
    fileId: string;
    expirySeconds?: number;
    disposition?: string;
  }) {
    const disposition = data.disposition === 'attachment' ? 'attachment' : 'inline';
    const result = await this.mediaService.getAccessUrl(
      data.fileId,
      disposition,
      data.expirySeconds || 900,
    );
    return {
      accessUrl: result.url,
      expiresAt: result.expiresAt,
      disposition: result.disposition,
    };
  }

  @GrpcMethod('MediaInternalService', 'GetPresignedDownloadUrl')
  async getPresignedDownloadUrl(data: { fileId: string; expirySeconds?: number }) {
    const result = await this.mediaService.getAccessUrl(
      data.fileId,
      'attachment',
      data.expirySeconds || 900,
    );
    return {
      downloadUrl: result.url,
      expiresAt: result.expiresAt,
    };
  }

  @GrpcMethod('MediaInternalService', 'BindMediaToEntity')
  async bindMediaToEntity(data: {
    fileId?: string;
    objectKey?: string;
    entityType: string;
    entityId: string;
  }) {
    const success = await this.mediaService.bindToEntity(
      data.fileId,
      data.objectKey,
      data.entityType,
      data.entityId,
    );
    return { success };
  }

  @GrpcMethod('MediaInternalService', 'PublishMedia')
  async publishMedia(data: { fileId?: string; objectKey?: string }) {
    return this.mediaService.publishMedia(data.fileId, data.objectKey);
  }
}
