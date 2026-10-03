FROM node:20-bookworm-slim AS build
WORKDIR /app
COPY package.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:20-bookworm-slim
WORKDIR /app
COPY --from=build /app/package*.json ./
RUN npm ci --omit=dev
COPY --from=build /app/.next ./.next
COPY --from=build /app/public ./public
RUN useradd --system --uid 10001 --create-home nextjs
USER nextjs
EXPOSE 3000
CMD ["npm", "run", "start"]
