# Stage 1: Dependencies
FROM node:22-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

# Stage 2: Development
FROM deps AS dev
WORKDIR /app
COPY . .
EXPOSE 5173
CMD ["npm", "run", "dev"]

# Stage 3: Build
FROM deps AS build
WORKDIR /app
COPY . .
RUN npm run build

# Stage 4: Production
FROM nginx:alpine-slim AS production
RUN addgroup -g 1001 -S appgroup && adduser -S appuser -u 1001 -G appgroup
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx/nginx.conf /etc/nginx/nginx.conf
RUN chown -R appuser:appgroup /usr/share/nginx/html /var/cache/nginx /var/log/nginx /etc/nginx/conf.d && \
    touch /var/run/nginx.pid && chown appuser:appgroup /var/run/nginx.pid
USER appuser
EXPOSE 8080
CMD ["nginx", "-g", "daemon off;"]
