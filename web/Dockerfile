# syntax=docker/dockerfile:1

FROM node:22.23.4-alpine AS deps
WORKDIR /src
RUN --mount=type=bind,source=package.json,target=package.json \
    --mount=type=bind,source=package-lock.json,target=package-lock.json \
    --mount=type=cache,target=/root/.npm \
    npm ci

FROM node:22.23.4-alpine AS build
WORKDIR /src
COPY --from=deps --link /src/node_modules ./node_modules
COPY . .
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

FROM node:22.23.4-alpine AS runtime
WORKDIR /app
ENV NODE_ENV=production \
    NEXT_TELEMETRY_DISABLED=1 \
    PORT=3847 \
    HOSTNAME=0.0.0.0
RUN addgroup --system --gid 1001 appgroup \
    && adduser --system --uid 1001 --ingroup appgroup appuser
COPY --from=build --link --chown=1001:1001 /src/public ./public
COPY --from=build --link --chown=1001:1001 /src/.next/standalone ./
COPY --from=build --link --chown=1001:1001 /src/.next/static ./.next/static
USER appuser
EXPOSE 3847
CMD ["node", "server.js"]
