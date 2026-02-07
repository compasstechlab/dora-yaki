# Build stage
FROM node:24-bookworm AS builder

RUN corepack enable && corepack prepare pnpm@latest --activate

WORKDIR /app

# Copy package files
COPY package.json pnpm-lock.yaml ./

# Install dependencies
RUN pnpm install --frozen-lockfile

# Copy source code
COPY . .

# Build-time environment variables (Vite will bundle them statically)
ARG VITE_API_BASE=/api
ARG VITE_DEFAULT_LOCALE=
ENV VITE_API_BASE=${VITE_API_BASE}
ENV VITE_DEFAULT_LOCALE=${VITE_DEFAULT_LOCALE}

# Build the application
RUN pnpm run build

# Production stage (distroless: minimal image)
FROM gcr.io/distroless/nodejs24-debian12

WORKDIR /app

# Copy built application
COPY --from=builder /app/build ./build
COPY --from=builder /app/package.json ./
COPY --from=builder /app/node_modules ./node_modules

# Set environment variables
ENV NODE_ENV=production
ENV PORT=7201

# Expose port
EXPOSE 7201

# distroless nodejs image has /nodejs/bin/node as entrypoint
CMD ["build/index.js"]
