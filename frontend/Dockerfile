# Multi-stage build for InfraSense frontend

# Build stage
FROM node:20.12-alpine AS builder

WORKDIR /app

# Set API URL (can be overridden at build time)
ARG VITE_API_URL=/api/v1
ENV VITE_API_URL=${VITE_API_URL}

# Copy package files and install dependencies
COPY package.json package-lock.json* ./
RUN npm install --legacy-peer-deps

# Copy source code and build
COPY . .
RUN NODE_OPTIONS="--max-old-space-size=2048" npm run build

# Runtime stage
FROM nginx:1.25-alpine

# Copy built static files
COPY --from=builder /app/dist /usr/share/nginx/html

# Copy nginx configuration
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
