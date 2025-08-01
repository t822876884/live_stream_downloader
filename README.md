# live_stream_downloader

## Docker构建

### 方式一
```shell
docker build -t live-stream-downloader .
```
```shell
docker run -d \
  --name live-stream-downloader \
  -p 18080:8080 \
  -v /var/services/homes/bertram/liveDown:/app/data \
  live-stream-downloader
```
### 方式二

```shell
docker-compose up -d
```

## Api请求

```http
curl --location 'http://localhost:8080/api/tasks' \
--header 'Content-Type: application/json' \
--data '{
    "url": "http://01yowzbola.wljzml.top/live/cx_354263.flv",
    "file_name": "希希.flv"
}'
```