# web-user — Next.js standalone build.
# Base image is a floating tag today; pin it by digest after the first
# successful Coolify build records one (W0/E4 follow-up).
FROM node:22-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

FROM node:22-alpine AS build
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
# NEXT_PUBLIC_* values are inlined at build time; pass them as build args.
ARG NEXT_PUBLIC_API_URL
ARG NEXT_PUBLIC_TERMS_VERSION
ARG NEXT_PUBLIC_MVP_COD_ENABLED
ARG NEXT_PUBLIC_MVP_VOUCHER_ENABLED
ARG NEXT_PUBLIC_MVP_GUEST_ORDER_ENABLED
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

FROM node:22-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
RUN addgroup -S nodejs && adduser -S nextjs -G nodejs
COPY --from=build /app/public ./public
COPY --from=build --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=build --chown=nextjs:nodejs /app/.next/static ./.next/static
USER nextjs
EXPOSE 3000
ENV PORT=3000
CMD ["node", "server.js"]
