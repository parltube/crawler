# crawler
crawler of youtube has a anti timeout, proxy usage, and etc. in it. mainly it gets metadata, comments, views and etc. 

# arch 

Буду писать на русском

Это кравлер для ютуба, собирает расданные по всем каналам которые сможет найти в отличии от filnot собирает полные данные с субтитрами и т.д. и отправляет через очередь в postgresql storage service-а. 

Как видит то что он уже там был? 

РасДанные - расшеренные данные про видео + канал 

Сервер 1: Kimsufi KS-1-B (~$15/мес, 8 ТБ HDD, 32 ГБ RAM)
├── PostgreSQL (данные + очереди)
├── Redis (кэш, maxmemory 2 ГБ)
├── Manticore (поисковый индекс, Tier 1-2 в RAM, ~6 ГБ)
├── Storage + Services — один C# модульный монолит
│   ├── Modules/
│   │   ├── Videos/          — CRUD видео, каналов
│   │   ├── Search/          — Typesense, каскадный поиск
│   │   ├── Recommendations/ — BM25, история
│   │   ├── Users/           — аккаунты, история, лайки
│   │   ├── Crawling/        — очередь задач, приём данных
│   │   └── Processing/      — фоновые воркеры
│   ├── Background Workers (IHostedService)
│   │   ├── SpamFilterWorker
│   │   ├── AiDetectWorker
│   │   ├── IndexSyncWorker
│   │   ├── TierClassifier
│   │   ├── ChannelAggregator
│   │   ├── StatsUpdater
│   │   ├── CacheInvalidator (event-driven)
│   │   ├── ArchiveWorker
│   │   ├── VacuumScheduler
│   │   ├── PartitionManager
│   │   ├── SearchWarmup
│   │   ├── HistoryImportWorker
│   │   ├── RecommendationPrecompute
│   │   ├── EmbeddingDispatcher
│   │   ├── HealthCheckWorker
│   │   ├── CrawlRateMonitor
│   │   └── StaleTaskReaper
│   └── Nginx (reverse proxy + rate limit)
└── WireGuard endpoint

Сервер 2: VPS ($5/мес)
├── Crawler × 4 инстанса (Python, async)
├── Mullvad VPN
└── WireGuard → подключение к PostgreSQL на сервере 1

Домашний PC: RTX 3080
└── Embedding worker (Python, через WireGuard)