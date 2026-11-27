# go-musthave-metrics-tpl

Шаблон репозитория для трека «Сервер сбора метрик и алертинга».

## Оптимизация памяти (pprof)

Проведено профилирование памяти функции сжатия метрик `compress` в пакете `internal/agent` с помощью `pprof`.
Для устранения избыточных аллокаций `gzip.Writer` внедрен пул объектов `sync.Pool`.

Результат сравнения профилей (`go tool pprof -top -diff_base=profiles/base.pprof profiles/result.pprof`):

File: agent.test.exe
Type: alloc_space
Showing nodes accounting for -11.11GB, 99.38% of 11.18GB total
Dropped 71 nodes (cum <= 0.06GB)
      flat  flat%   sum%        cum   cum%
   -6.12GB 54.72% 54.72%   -11.11GB 99.30%  compress/flate.NewWriter (inline)
   -3.04GB 27.22% 81.94%    -4.99GB 44.58%  compress/flate.(*compressor).init
   -1.91GB 17.05% 98.99%    -1.91GB 17.05%  compress/flate.newDeflateFast (inline)
   -0.06GB  0.54% 99.53%    -0.06GB  0.54%  compress/flate.(*huffmanEncoder).generate
    0.02GB  0.15% 99.38%   -11.10GB 99.26%  github.com/PRoroKEd1/go-metrics/internal/agent.compress
         0     0% 99.38%    -0.06GB  0.54%  compress/flate.(*compressor).encSpeed
         0     0% 99.38%    -0.06GB  0.54%  compress/flate.(*huffmanBitWriter).writeBlockDynamic
         0     0% 99.38%   -11.08GB 99.11%  compress/gzip.(*Writer).Write
         0     0% 99.38%   -11.10GB 99.26%  github.com/PRoroKEd1/go-metrics/internal/agent.BenchmarkCompress
         0     0% 99.38%   -11.10GB 99.25%  testing.(*B).launch
         0     0% 99.38%   -11.10GB 99.27%  testing.(*B).runN