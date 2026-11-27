FROM node:20-alpine

WORKDIR /app

# sharp native libvips dependencies on Alpine
RUN apk add --no-cache vips-dev

COPY package*.json ./
RUN npm ci --only=production && npm cache clean --force

COPY dist ./dist
COPY prisma ./prisma
RUN npx prisma generate

RUN addgroup -S appgroup && adduser -S appuser -G appgroup \
    && chown -R appuser:appgroup /app

USER appuser

EXPOSE 8007 9007

CMD ["node", "dist/main.js"]
