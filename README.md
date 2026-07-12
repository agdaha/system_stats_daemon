# "Системный мониторинг"

---

# ТЗ на разработку демона, осуществляющего "Системный мониторинг"

[](https://github.com/OtusGolang/final_project/blob/master/05-system-stats-daemon.md#тз-на-разработку-демона-осуществляющего-системный-мониторинг)

## Общее описание

[](https://github.com/OtusGolang/final_project/blob/master/05-system-stats-daemon.md#общее-описание)

Демон - программа, собирающая информацию о системе, на которой запущена,
и отправляющая её своим клиентам по GRPC.

## Архитектура

[](https://github.com/OtusGolang/final_project/blob/master/05-system-stats-daemon.md#архитектура)

* GRPC сервер;
* допускается использование временных (`/tmp`) файлов;
* статистика хранится в памяти, долговременное хранение не предусмотрено.

## Требования

[](https://github.com/OtusGolang/final_project/blob/master/05-system-stats-daemon.md#требования)

Необходимо каждые **N** секунд выдавать информацию, усредненную за последние **M** секунд.

Например, N = 5с, а M = 15с, тогда демон "молчит" первые 15 секунд,
затем выдает снапшот за 0-15с; через 5с (в 20с) выдает снапшот за 5-20с;
через 5с (в 25с) выдает снапшот за 10-25с и т.д.

**N** и **M** указывает клиент в запросе на получение статистики.

Что необходимо собирать:

* Средняя загрузка системы (load average).
* Средняя загрузка CPU (%user_mode, %system_mode, %idle).
* Загрузка дисков:
  * tps (transfers per second);
  * KB/s (kilobytes (read+write) per second);
* Информация о дисках по каждой файловой системе:
  * использовано мегабайт, % от доступного количества;
  * использовано inode, % от доступного количества.
* Top talkers по сети:
  * по протоколам: protocol (TCP, UDP, ICMP, etc), bytes, % от sum(bytes) за последние  **M** ), сортируем по убыванию процента;
  * по трафику: source ip:port, destination ip:port, protocol, bytes per second (bps), сортируем по убыванию bps.
* Статистика по сетевым соединениям:
  * слушающие TCP & UDP сокеты: command, pid, user, protocol, port;
  * количество TCP соединений, находящихся в разных состояниях (ESTAB, FIN_WAIT, SYN_RCV и пр.).

#### Разрешено использовать только стандартную библиотеку языка Go!

[](https://github.com/OtusGolang/final_project/blob/master/05-system-stats-daemon.md#разрешено-использовать-только-стандартную-библиотеку-языка-go)

Команды, которые могут пригодиться:

```
$ top -b -n1
$ df -k
$ df -i
$ iostat -d -k
$ cat /proc/net/dev
$ sudo netstat -lntup
$ ss -ta
$ tcpdump -ntq -i any -P inout -l
$ tcpdump -nt -i any -P inout -ttt -l
```

Статистика представляет собой объекты, описанные в формате Protobuf.

Информацию необходимо выдавать всем подключенным по GRPC клиентам
с использованием [однонаправленного потока](https://grpc.io/docs/tutorials/basic/go/#server-side-streaming-rpc).

Выдавать "снапшот" системы можно как отдельными сообщениями, так и одним жирным объектом.

Сбор информации, её парсинг и пр. должен осуществляться как можно более конкурентно.

## Поддерживаемая ОС

[](https://github.com/OtusGolang/final_project/blob/master/05-system-stats-daemon.md#поддерживаемая-ос)

Минимум - Linux (Ubuntu 18.04).

Максимум - несколько сборок под набор из популярных ОС/процессоров:

* darwin, linux, windows
* 386, amd64

[Список возможных вариантов](https://gist.github.com/asukakenji/f15ba7e588ac42795f421b48b8aede63).

Но тогда придется постараться с реализацией использования различных команд для сбора данных.

Пригодятся [build тэги](https://www.digitalocean.com/community/tutorials/building-go-applications-for-different-operating-systems-and-architectures).

## Конфигурация

[](https://github.com/OtusGolang/final_project/blob/master/05-system-stats-daemon.md#конфигурация)

* Через аргументы командной строки можно указать, на каком порту стартует сервер.
* Через файл можно указать, какие из подсистем сбора включены/выключены.

## Тестирование

[](https://github.com/OtusGolang/final_project/blob/master/05-system-stats-daemon.md#тестирование)

#### Юнит-тесты

[](https://github.com/OtusGolang/final_project/blob/master/05-system-stats-daemon.md#юнит-тесты)

* по возможности мок интерфейсов и проверка вызовов конкретных методов;
* тесты вспомогательных функций и пр.

#### Интеграционные тесты

[](https://github.com/OtusGolang/final_project/blob/master/05-system-stats-daemon.md#интеграционные-тесты)

* потестировать факт потока статистики, можно без конкретных цифр;
* можно посоздавать файлы, пооткрывать сокеты и посмотреть на изменение снапшота.

#### Клиент

[](https://github.com/OtusGolang/final_project/blob/master/05-system-stats-daemon.md#клиент)

Необходимо реализовать простой клиент, который в реальном времени получает
и выводит в STDOUT статистику по одному из пунктов (например, сетевую информацию)
в читаемом формате (например, в виде таблицы).

## Разбалловка

[](https://github.com/OtusGolang/final_project/blob/master/05-system-stats-daemon.md#разбалловка)

Максимум - **20 баллов**
(при условии выполнения [обязательных требований](https://github.com/OtusGolang/final_project/blob/master/README.md)):

* Реализован сбор:
  * load average - 1 балл;
  * загрузка CPU - 1 балл;
  * загрузка дисков - 1 балл;
  * top talkers по сети - 1 балла;
  * статистика по сети - 1 балл.
* Через конфигурацию можно отключать отдельную статистику - 2 балла.
* Написаны юнит-тесты - 1 балл.
* Написаны интеграционные тесты - 2 балла.
* Реализован простой клиент к демону - 2 балла.
* Сбор хотя бы одного типа статистики работает на разных ОС - 5 баллов.
* Понятность и чистота кода - до 3 баллов.

---

# Реализация

## Быстрый старт

**Требования:** Go ≥ 1.22, `protoc` (только для регенерации proto), `golangci-lint`.

```bash
# Сборка
make build          # → bin/sysmon-daemon, bin/sysmon-client

# Запуск демона (порт по умолчанию 50051)
./bin/sysmon-daemon

# Запуск демона на другом порту с конфигом
./bin/sysmon-daemon --port 50051 --config config.json

# Подключение клиента
./bin/sysmon-client --addr localhost:50051 --n 5 --m 15

# Только конкретные подсистемы
./bin/sysmon-client --addr localhost:50051 --n 5 --m 15 --subsystem loadavg,cpu
```

После запуска клиент молчит первые M секунд (накопление буфера), затем выводит обновляемый снапшот в терминал.

## Конфигурация демона

### CLI-флаги

| Флаг     | По умолчанию | Описание                                     |
| ------------ | ----------------------- | ---------------------------------------------------- |
| `--port`   | `50051`               | Порт gRPC-сервера                         |
| `--config` | —                      | Путь к JSON-файлу конфигурации |

### Файл конфигурации (JSON)

По умолчанию все подсистемы включены. Файл позволяет выборочно отключить любую:

```json
{
  "subsystems": {
    "load_average": true,
    "cpu": true,
    "disk_io": true,
    "filesystem": true,
    "net_traffic": false,
    "net_sockets": false
  }
}
```

> `net_traffic` использует `tcpdump` и требует прав root или `CAP_NET_RAW`.
> `net_sockets` использует `netstat` и может требовать root для отображения PID/команды.

## CLI-клиент

### Флаги клиента

| Флаг        | По умолчанию | Описание                                          |
| --------------- | ----------------------- | --------------------------------------------------------- |
| `--addr`      | `localhost:9090`      | Адрес демона                                   |
| `--n`         | `5`                   | Интервал снапшота в секундах     |
| `--m`         | `15`                  | Окно усреднения в секундах         |
| `--subsystem` | _(все)_            | Фильтр подсистем через запятую |

### Допустимые значения `--subsystem`

`loadavg`, `cpu`, `diskio`, `filesystem`, `nettraffic`, `netsockets`

## Запуск тестов

```bash
make test                   # юнит-тесты: go test -race -count 100 ./...
make test-integration       # интеграционные тесты (только Linux, требует реальной ОС)
make lint                   # golangci-lint
```

Интеграционные тесты поднимают настоящий gRPC-сервер на случайном loopback-порту
и проверяют e2e-поток снапшотов. Требуют тег `integration`:

```bash
go test -race -tags integration -timeout 120s ./tests/integration/...
```

## Docker

```bash
docker build -t sysmon-daemon .
docker run --rm -p 50051:50051 sysmon-daemon
```

Для корректной работы `net_traffic` (tcpdump) контейнер нужно запускать с `--cap-add NET_RAW --net host`.

## CI/CD (GitHub Actions)

Пайплайн содержит четыре джоба:

| Джоб             | Что делает                                                             |
| -------------------- | ------------------------------------------------------------------------------- |
| `lint`             | `golangci-lint` с конфигом из репозитория               |
| `test`             | `go test -race -count 100 ./...`                                              |
| `integration-test` | интеграционные тесты на ubuntu-latest                      |
| `build`            | кросс-компиляция для linux/amd64, darwin/amd64, windows/amd64 |

## Архитектура реализации

```
cmd/
  daemon/main.go      — точка входа демона (флаги, старт коллекторов, gRPC-сервер)
  client/main.go      — CLI-клиент (флаги, --subsystem фильтр, табличный вывод)
internal/
  config/             — загрузка CLI-флагов и JSON-конфига
  server/             — gRPC server-streaming RPC, агрегация по окну M
  collector/
    loadavg/          — /proc/loadavg, 1 с
    cpu/              — /proc/stat (diff), 1 с
    diskio/           — /proc/diskstats (diff), 1 с
    filesystem/       — df -kP / df -iP, 5 с
    nettraffic/       — /proc/net/dev (diff) + tcpdump, 1–5 с
    netsockets/       — ss -ta / netstat -lntup, 5 с
api/stats/            — сгенерированный protobuf/gRPC код
tests/integration/    — e2e-тесты (тег integration && linux)
```

Каждый коллектор независимо пишет замеры в кольцевой буфер (`RingBuffer`).
Сервер при получении запроса от клиента читает из буферов и усредняет данные за последние M секунд,
затем отдаёт снапшот каждые N секунд. Все подсистемы выбираются через интерфейсы — коллекторы
заменяемы моками в тестах без изменения кода сервера.

Платформозависимые реализации разделены build-тегами (`//go:build linux`);
для остальных ОС — заглушки `_other.go`, возвращающие пустые данные.
