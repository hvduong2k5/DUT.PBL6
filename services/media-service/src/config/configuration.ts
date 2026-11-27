export default () => ({
  httpPort: parseInt(process.env.HTTP_PORT || '8007', 10),
  grpcPort: parseInt(process.env.GRPC_PORT || '9007', 10),
  minio: {
    endpoint: process.env.MINIO_ENDPOINT || 'http://localhost:9000',
    publicEndpoint: process.env.MINIO_PUBLIC_ENDPOINT || process.env.MINIO_ENDPOINT || 'http://localhost:9000',
    accessKey: process.env.MINIO_ACCESS_KEY || 'minioadmin',
    secretKey: process.env.MINIO_SECRET_KEY || 'minioadmin',
    region: process.env.MINIO_REGION || 'us-east-1',
    buckets: {
      catalog: process.env.MEDIA_BUCKET_CATALOG || 'om-catalog',
      traceability: process.env.MEDIA_BUCKET_TRACEABILITY || 'om-traceability',
      fulfillment: process.env.MEDIA_BUCKET_FULFILLMENT || 'om-fulfillment',
      documents: process.env.MEDIA_BUCKET_DOCUMENTS || 'om-documents',
      private: process.env.MEDIA_BUCKET_PRIVATE || 'om-private',
    },
  },
  kafka: {
    enabled: process.env.KAFKA_ENABLED === 'true',
    brokers: (process.env.KAFKA_BROKERS || 'localhost:9092').split(','),
    clientId: process.env.KAFKA_CLIENT_ID || 'media-service',
    topic: process.env.KAFKA_TOPIC_MEDIA_EVENTS || 'media.events.v1',
  },
  presignedPutExpirySeconds: parseInt(process.env.MEDIA_PRESIGNED_PUT_EXPIRY_SECONDS || '900', 10),
  presignedGetExpirySeconds: parseInt(process.env.MEDIA_PRESIGNED_GET_EXPIRY_SECONDS || '900', 10),
  cdnBaseUrl: process.env.MEDIA_CDN_BASE_URL || 'http://localhost:9000/om-catalog/',
});
