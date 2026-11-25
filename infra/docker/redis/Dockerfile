FROM redis:7.2-alpine

ENV REDIS_PASSWORD=redispassword

EXPOSE 6379

CMD ["sh", "-c", "exec redis-server --appendonly yes --requirepass \"${REDIS_PASSWORD}\""]
