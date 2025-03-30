FROM node:20-alpine AS builder
WORKDIR /app
COPY package.json ./
RUN npm install
COPY . .
RUN npm run build

FROM alpine:3.23.3
COPY --from=builder /app/dist /dist
CMD ["sh", "-c", "echo 'Admin static files built' && exit 0"]