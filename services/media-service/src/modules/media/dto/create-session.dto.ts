import { IsEnum, IsInt, IsOptional, IsString, Max, MaxLength, Min } from 'class-validator';
import { MediaPurpose } from '@prisma/client';

export class CreateUploadSessionDto {
  @IsEnum(MediaPurpose)
  purpose!: MediaPurpose;

  @IsString()
  @MaxLength(500)
  fileName!: string;

  @IsString()
  @MaxLength(128)
  mediaType!: string;

  @IsInt()
  @Min(1)
  @Max(104857600) // Max 100MB
  sizeBytes!: number;

  @IsOptional()
  @IsString()
  @MaxLength(64)
  sha256?: string;
}
