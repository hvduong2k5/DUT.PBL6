import { ConfigService } from '@nestjs/config';
import { AccessLevel, MediaPurpose, MediaStatus } from '@prisma/client';
import { MediaService } from './media.service';

describe('MediaService', () => {
  let service: MediaService;
  let prismaMock: any;
  let storageMock: any;
  let kafkaMock: any;
  let optimizerMock: any;
  let configMock: any;

  beforeEach(() => {
    prismaMock = {
      mediaFile: {
        create: jest.fn(),
        findUnique: jest.fn(),
        update: jest.fn(),
      },
      uploadSession: {
        create: jest.fn(),
        findUnique: jest.fn(),
        update: jest.fn(),
      },
      $transaction: jest.fn(async (ops: any[]) => Promise.all(ops)),
    };

    storageMock = {
      getPresignedPutUrl: jest.fn().mockResolvedValue('http://localhost:9000/presigned-put'),
      getPresignedGetUrl: jest.fn().mockResolvedValue('http://localhost:9000/presigned-get'),
      headObject: jest.fn().mockResolvedValue({ contentLength: 2048 }),
      copyObject: jest.fn().mockResolvedValue(undefined),
    };

    kafkaMock = {
      emitEvent: jest.fn().mockResolvedValue(undefined),
    };

    optimizerMock = {
      processFile: jest.fn().mockResolvedValue(undefined),
    };

    configMock = {
      get: jest.fn((key: string) => {
        const map: Record<string, any> = {
          'minio.buckets.private': 'om-private',
          'minio.buckets.catalog': 'om-catalog',
          'minio.buckets.traceability': 'om-traceability',
          'minio.buckets.fulfillment': 'om-fulfillment',
          'minio.buckets.documents': 'om-documents',
          presignedPutExpirySeconds: 900,
          presignedGetExpirySeconds: 900,
        };
        return map[key];
      }),
    };

    service = new MediaService(
      prismaMock,
      storageMock,
      kafkaMock,
      optimizerMock,
      configMock as unknown as ConfigService,
    );
  });

  it('generates objectKey following {entity-type}/{YYYY}/{MM}/{DD}/{uuid}.{ext}', () => {
    const key = service.generateObjectKey(
      MediaPurpose.CATALOG_PRODUCT,
      'anh me xung.jpg',
      '9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d',
    );
    expect(key).toMatch(/^products\/\d{4}\/\d{2}\/\d{2}\/9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d\.jpg$/);
  });

  it('routes unapproved CATALOG_PRODUCT uploads to private bucket om-private', async () => {
    prismaMock.mediaFile.create.mockResolvedValue({});
    prismaMock.uploadSession.create.mockResolvedValue({});

    const res = await service.createUploadSession(
      {
        purpose: MediaPurpose.CATALOG_PRODUCT,
        fileName: 'candy.jpg',
        mediaType: 'image/jpeg',
        sizeBytes: 2048,
      },
      'staff-1',
    );

    expect(res.bucket).toBe('om-private');
    expect(prismaMock.mediaFile.create).toHaveBeenCalledWith(
      expect.objectContaining({
        data: expect.objectContaining({
          bucketName: 'om-private',
          accessLevel: AccessLevel.PRIVATE_RESTRICTED,
          status: MediaStatus.AWAITING_UPLOAD,
        }),
      }),
    );
  });

  it('transitions session to PROCESSING on complete and emits file.uploaded.v1', async () => {
    const mockFile = {
      id: 'file-1',
      objectKey: 'products/2026/11/25/file-1.jpg',
      bucketName: 'om-private',
      fileName: 'candy.jpg',
      mediaType: 'image/jpeg',
      sizeBytes: BigInt(2048),
      sha256Checksum: null,
      purpose: MediaPurpose.CATALOG_PRODUCT,
      accessLevel: AccessLevel.PRIVATE_RESTRICTED,
      status: MediaStatus.PROCESSING,
      variantsMeta: {},
      createdAt: new Date(),
      updatedAt: new Date(),
    };

    prismaMock.uploadSession.findUnique.mockResolvedValue({
      id: 'sess_1',
      fileId: 'file-1',
      status: 'ACTIVE',
      file: { ...mockFile, status: MediaStatus.AWAITING_UPLOAD },
    });
    prismaMock.mediaFile.update.mockResolvedValue(mockFile);
    prismaMock.uploadSession.update.mockResolvedValue({ id: 'sess_1', status: 'COMPLETED' });

    const res = await service.completeUploadSession('sess_1');
    expect(res.status).toBe(MediaStatus.PROCESSING);
    expect(kafkaMock.emitEvent).toHaveBeenCalledWith(
      'vn.omama.media.file.uploaded.v1',
      mockFile.objectKey,
      expect.objectContaining({ fileId: 'file-1' }),
    );
  });

  it('promotes approved media from om-private to om-catalog on publishMedia', async () => {
    prismaMock.mediaFile.findUnique.mockResolvedValue({
      id: 'file-1',
      objectKey: 'products/2026/11/25/file-1.jpg',
      bucketName: 'om-private',
      variantsMeta: {
        webp: 'products/2026/11/25/file-1.webp',
        thumb_150: 'products/2026/11/25/file-1_150.webp',
      },
    });
    prismaMock.mediaFile.update.mockResolvedValue({});

    const res = await service.publishMedia('file-1');
    expect(res.success).toBe(true);
    expect(storageMock.copyObject).toHaveBeenCalledTimes(3);
    expect(prismaMock.mediaFile.update).toHaveBeenCalledWith(
      expect.objectContaining({
        data: {
          bucketName: 'om-catalog',
          accessLevel: AccessLevel.PUBLIC_READ,
        },
      }),
    );
  });
});
