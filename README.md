# media-service

Multi-thread go app for processing media files from remote url

> **Task definition (from customer):**
> 
> Требуется написать на go сервис, который получая запросы вида: `http://localhost:8080/dl/?url=http://78.140.172.155/test/sintel.mp4&md5=c689c2d468f841a20116992032dc09ca` скачивает файл, проверяет его md5, в зависимости от вида произошедшей ошибки, делает несколько попыток перезакачки.
> 
> Затем через `ffmpeg` узнаёт битрейт и разрешение видео, сохраняет эту информацию.
> 
> Можно использовать `sqlite`, или любое удобное простое хранилище.
> 
> Все попытки совершения операций, успешность или ошибочность, следует писать в лог. Должна быть возможность через URL типа `http://localhost:8080/st/` посмотреть нынешний статус любого запроса, прогресс, ошибки.
> 
> Если сервис был перезагружен (например, вместе с сервером, допустим после внезапного падения) сервис должен адекватно восстановить работоспособность.

## How it work:

- collect tasks (HTTP GET request)
- download remote media file
- validate checksum and get media file info
- load info to storage (sqLite). Can be simply replaced (S3, local filesystem, whatever)

## How use it:

You can start up Virtual Machine (if you want):

```bash 
vagrant up
vagrant ssh
```

### Run as local binary

```bash
make build && bash launch.sh run
```

### Run as docker container

```bash
make docker && bash launch.sh run-docker
```

### For testing

For local testing run in **other terminal**:

```bash
# send one request for downloading
bash launch.sh test-light

# send some requests for downloading
bash launch.sh test-heavy

# show statistics about downloads (JSON)
bash launch.sh test-web

# for more information how use see launch.sh:
bash launch.sh
```

### If you want develop

```bash
# fast recompile and run
# but remember: you must before building source if not yet (make build)
make recompile && bash launch.sh run
```
