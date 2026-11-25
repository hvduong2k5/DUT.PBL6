import {
  Body,
  Controller,
  Delete,
  Get,
  Headers,
  Param,
  Post,
  Query,
  UploadedFile,
  UseInterceptors,
} from '@nestjs/common';
import { FileInterceptor } from '@nestjs/platform-express';
import { MediaPurpose } from '@prisma/client';
import { CreateUploadSessionDto } from './dto/create-session.dto';
import { MediaService } from './media.service';

@Controller('api/v1/media')
export class MediaController {
  constructor(private readonly mediaService: MediaService) {}

  @Post('upload-sessions')
  createUploadSession(
    @Body() dto: CreateUploadSessionDto,
    @Headers('x-user-id') userId = 'anonymous',
  ) {
    return this.mediaService.createUploadSession(dto, userId);
  }

  @Post('upload-sessions/:id/complete')
  completeUploadSession(@Param('id') sessionId: string) {
    return this.mediaService.completeUploadSession(sessionId);
  }

  @Get(':fileId')
  getFile(@Param('fileId') fileId: string) {
    return this.mediaService.getFile(fileId);
  }

  @Get(':fileId/view-url')
  getViewUrl(
    @Param('fileId') fileId: string,
    @Query('expirySeconds') expirySeconds?: string,
  ) {
    return this.mediaService.getAccessUrl(
      fileId,
      'inline',
      expirySeconds ? parseInt(expirySeconds, 10) : 900,
    );
  }

  @Get(':fileId/download-url')
  getDownloadUrl(
    @Param('fileId') fileId: string,
    @Query('expirySeconds') expirySeconds?: string,
  ) {
    return this.mediaService.getAccessUrl(
      fileId,
      'attachment',
      expirySeconds ? parseInt(expirySeconds, 10) : 900,
    );
  }

  @Post('direct-upload')
  @UseInterceptors(FileInterceptor('file'))
  directUpload(
    @UploadedFile() file: Express.Multer.File,
    @Body('purpose') purpose: MediaPurpose = MediaPurpose.CATALOG_PRODUCT,
    @Headers('x-user-id') userId = 'admin',
  ) {
    return this.mediaService.directUpload(file, purpose, userId);
  }

  @Delete(':fileId')
  softDelete(@Param('fileId') fileId: string) {
    return this.mediaService.softDelete(fileId);
  }
}
