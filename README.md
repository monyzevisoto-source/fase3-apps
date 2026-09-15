# Fase 3 — Aplicações ToggleMaster

Monorepo com os cinco microsserviços da plataforma. Cada serviço é independente,
possui seu próprio código, Dockerfile, dependências, workflow de CI/CD e imagem
no Amazon ECR.

## Serviços

| Serviço | Tecnologia | Responsabilidade | Porta |
| --- | --- | --- | --- |
| auth-service | Go | Criação e validação de chaves de API | 8001 |
| flag-service | Python/Flask | CRUD das definições de feature flags | 8002 |
| targeting-service | Python/Flask | Regras de segmentação das flags | 8003 |
| evaluation-service | Go | Avaliação rápida das flags com cache Redis | 8004 |
| analytics-service | Python | Consumo de eventos SQS e gravação no DynamoDB | 8005 |

## Como os serviços funcionam

O evaluation-service recebe as avaliações dos clientes. Em caso de cache miss,
consulta flag-service e targeting-service, grava o resultado no Redis, responde
true ou false e publica um evento no SQS. O analytics-service consome esse evento
e grava os dados na tabela DynamoDB ToggleMasterAnalytics.

O auth-service protege as APIs administrativas. flag-service e targeting-service
usam PostgreSQL e exigem uma chave de API válida. Os detalhes de endpoints e
variáveis de ambiente estão no README de cada serviço.

## Estrutura

`text
.
├── auth-service/
├── flag-service/
├── targeting-service/
├── evaluation-service/
├── analytics-service/
├── .github/workflows/
│   ├── auth-ci.yml
│   ├── flag-ci.yml
│   ├── targeting-ci.yml
│   ├── evaluation-ci.yml
│   └── analytics-ci.yml
└── docs/CICD.png
`

Cada diretório de serviço contém código-fonte, Dockerfile, dependências, schema
db/init.sql quando aplicável e README específico. O diretório .github/workflows
contém um pipeline independente por serviço.

## Status dos builds

Os indicadores mostram o último resultado do workflow na branch main:

| Microsserviço | Build |
| --- | --- |
| auth-service | [![Auth CI](https://github.com/monyzevisoto-source/fase3-apps/actions/workflows/auth-ci.yml/badge.svg?branch=main)](https://github.com/monyzevisoto-source/fase3-apps/actions/workflows/auth-ci.yml) |
| flag-service | [![Flag CI](https://github.com/monyzevisoto-source/fase3-apps/actions/workflows/flag-ci.yml/badge.svg?branch=main)](https://github.com/monyzevisoto-source/fase3-apps/actions/workflows/flag-ci.yml) |
| targeting-service | [![Targeting CI](https://github.com/monyzevisoto-source/fase3-apps/actions/workflows/targeting-ci.yml/badge.svg?branch=main)](https://github.com/monyzevisoto-source/fase3-apps/actions/workflows/targeting-ci.yml) |
| evaluation-service | [![Evaluation CI](https://github.com/monyzevisoto-source/fase3-apps/actions/workflows/evaluation-ci.yml/badge.svg?branch=main)](https://github.com/monyzevisoto-source/fase3-apps/actions/workflows/evaluation-ci.yml) |
| analytics-service | [![Analytics CI](https://github.com/monyzevisoto-source/fase3-apps/actions/workflows/analytics-ci.yml/badge.svg?branch=main)](https://github.com/monyzevisoto-source/fase3-apps/actions/workflows/analytics-ci.yml) |

Verde indica sucesso e vermelho indica falha. Clique no badge para ver a
execução detalhada.

## Diagrama do CI/CD

![Fluxo CI/CD](docs/CICD.png)

## Desenvolvimento local

Para Go:

`bash
cd auth-service                 # ou evaluation-service
go mod download
go build ./...
go test ./...
go run .
`

Para Python:

`bash
cd flag-service                 # ou targeting-service/analytics-service
python -m pip install -r requirements.txt
python -m compileall .
gunicorn --bind 0.0.0.0:8002 app:app
`

Use a porta correspondente ao serviço. Inicialize os bancos PostgreSQL com o
arquivo db/init.sql de auth-service, flag-service e targeting-service.

## CI/CD e publicação

Um push na main executa o workflow do serviço alterado pelos filtros de caminho.
Uma tag Git no formato vMAJOR.MINOR.PATCH executa os cinco workflows.

Cada pipeline executa build, testes, lint, análise de segurança, Trivy, build da
imagem e scan da imagem. Em publicação, a imagem recebe a tag
vMAJOR.MINOR.PATCH-<sha de 7 caracteres> e é enviada ao ECR.

Depois do push, o job GitOps atualiza somente o Deployment correspondente no
repositório fase3-argocd e cria um commit deploy: update .... Os cinco jobs
GitOps usam uma fila compartilhada para evitar conflitos entre commits.

Configure o secret ARGOCD_REPO_TOKEN nas Actions deste repositório com permissão
Contents: Read and write no repositório fase3-argocd.
