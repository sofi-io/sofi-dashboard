# SOFI Dashboard

Implementação operacional e pública do Open Football Extractor (OFE), iniciativa do Open Football Compliance (OFC).

## MVP

O primeiro piloto cobre o Campeonato Pernambucano 2026/2027:

1. descobrir competição, temporada, clubes e partidas;
2. localizar e preservar borderôs oficiais;
3. submeter PDFs à SOFI API;
4. acompanhar extrações e persistir resultados brutos;
5. apresentar dados em interfaces interna e pública.

Nesta fase, o Dashboard consome a estrutura de extração já exposta pela SOFI API. Não implementamos normalização financeira OFE nem classificação de rubricas.

## Arquitetura

- Go: backend modular com dois executáveis (`cmd/api` e `cmd/worker`).
- Next.js: frontend em `web/`.
- PostgreSQL: metadados e resultados.
- MinIO/S3: documentos originais.
- SOFI API: serviço externo de extração autenticado server-to-server.

## Requisitos

- Go 1.23+
- Docker e Docker Compose
- Node.js LTS (para o frontend, quando iniciado)

## Executar localmente

```bash
cp .env.example .env
docker compose up -d postgres minio
go mod tidy
go run ./cmd/api
```

Em outro terminal:

```bash
go run ./cmd/worker
```

API: `http://localhost:8080/health/live` e `http://localhost:8080/health/ready`.

O worker nesta primeira entrega apenas inicializa, valida a conexão com o banco e aguarda encerramento. O processamento de filas será implementado nas próximas etapas.

## Migrations

As migrations são SQL versionadas em `migrations/`. Para esta primeira etapa, execute `migrations/000001_initial_schema.up.sql` no banco local. A automação de migrations será adicionada quando definirmos o fluxo de deploy.

## Segurança

- A chave SOFI API fica apenas no backend/worker.
- Não inserir segredos reais no Git.
- O browser nunca chama a SOFI API diretamente.
- A publicação pública dos documentos/resultados será explícita, não automática.
