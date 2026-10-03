FROM node:20-bookworm-slim AS build
WORKDIR /app
COPY package.json ./
RUN npm install --no-audit --no-fund
COPY . .
RUN npm run build

FROM node:20-bookworm-slim
WORKDIR /app
COPY --from=build /app/package*.json ./
RUN npm install --omit=dev --no-audit --no-fund
COPY --from=build /app/.next ./.next
COPY --from=build /app/public ./public
RUN useradd --system --uid 10001 --create-home nextjs
USER nextjs
EXPOSE 3000
CMD ["npm", "run", "start"]
