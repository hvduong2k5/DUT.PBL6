FROM node:20-alpine

WORKDIR /app

# Build context: repository root. Compile services/media-service/dist first.
# sharp native libvips dependencies on Alpine
RUN apk add --no-cache vips-dev
RUN apk add --no-cache openssl

COPY services/media-service/package*.json ./
# The locked Prisma CLI is needed for local migrations; never let npx fetch a different major.
RUN npm ci && npm cache clean --force

COPY services/media-service/dist ./dist
COPY services/media-service/prisma ./prisma
COPY packages/proto ./packages/proto
RUN npx prisma generate

RUN addgroup -S appgroup && adduser -S appuser -G appgroup \
    && mkdir -p /home/appuser

USER appuser

ENV PROTO_ROOT=/app/packages/proto

EXPOSE 8007 9007

CMD ["node", "dist/main.js"]
