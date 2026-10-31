# flag-service (Python)

Este é o serviço de CRUD (Create, Read, Update, Delete) do projeto ToggleMaster. Ele é responsável por gerenciar as *definições* das feature flags.

**IMPORTANTE:** Este serviço é protegido e depende que o `auth-service` esteja rodando. Todas as requisições (exceto `/health`) exigem um header `Authorization: Bearer <sua-chave-api>`.

## 📦 Pré-requisitos (Local)

* [Python](https://www.python.org/) (versão 3.9 ou superior)
* [PostgreSQL](https://www.postgresql.org/download/) (rodando localmente ou em um contêiner Docker)
* O `auth-service` deve estar rodando (localmente na porta `8001`).

## 🚀 Rodando Localmente

1.  **Clone o repositório** e entre na pasta `flag-service`.

2.  **Prepare o Banco de Dados:**
    * Crie um banco de dados no seu PostgreSQL (ex: `flags_db`).
    * Execute o script `db/init.sql` para criar a tabela `flags`:
        ```bash
        psql -U seu_usuario -d flags_db -f db/init.sql
        ```

3.  **Configure as Variáveis de Ambiente:**
    Crie um arquivo chamado `.env` na raiz desta pasta (`flag-service/`) com o seguinte conteúdo:
    ```.env
    # String de conexão do seu banco de dados PostgreSQL
    DATABASE_URL="postgres://SEU_USUARIO:SUA_SENHA@localhost:5432/flags_db"
    
    # Porta que este serviço (flag-service) irá rodar
    PORT="8002"
    
    # URL do auth-service (que deve estar rodando na porta 8001)
    AUTH_SERVICE_URL="http://localhost:8001"
    ```

4.  **Instale as Dependências:**
    ```bash
    pip install -r requirements.txt
    ```

5.  **Inicie o Serviço:**
    ```bash
    gunicorn --bind 0.0.0.0:8002 app:app
    ```
    O servidor estará rodando em `http://localhost:8002`.

## 🧪 Testando os Endpoints

**Primeiro, você precisa de uma chave de API válida!**

1.  Vá até o terminal do `auth-service` (que deve estar rodando) e crie uma chave:
    ```bash
    curl -X POST http://localhost:8001/admin/keys \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer admin-secreto-123" \
    -d '{"name": "admin-para-flag-service"}'
    ```
2.  Copie a chave retornada (ex: `tm_key_...`). Vamos chamá-la de `SUA_CHAVE_API` no resto dos exemplos.

---

**Agora, teste o `flag-service`:**

**1. Verifique a Saúde (Health Check):**
```bash
curl http://localhost:8002/health
```

Saída esperada: `{"status":"ok"}`

**2. Tente Acessar um Endpoint Protegido (Sem Chave):**
```bash
curl http://localhost:8002/flags
```

Saída esperada: `{"error":"Authorization header obrigatório"}`

**3. Crie uma nova Flag (Com a Chave Correta):**
```bash
curl -X POST http://localhost:8002/flags \
-H "Content-Type: application/json" \
-H "Authorization: Bearer SUA_CHAVE_API" \
-d '{
    "name": "enable-new-dashboard",
    "description": "Ativa o novo dashboard para usuários",
    "is_enabled": true
}'
```
Saída esperada: (Um JSON com os dados da flag criada).

**4. Liste todas as Flags:**
```bash
curl http://localhost:8002/flags \
-H "Authorization: Bearer SUA_CHAVE_API"
```
Saída esperada: (Uma lista `[]` contendo a flag que você criou).

**5. Desative a Flag (PUT):**
```bash
curl -X PUT http://localhost:8002/flags/enable-new-dashboard \
-H "Content-Type: application/json" \
-H "Authorization: Bearer SUA_CHAVE_API" \
-d '{"is_enabled": false}'
```
Saída esperada: (O JSON da flag atualizada, com `"is_enabled": false`).
## Logs via OTLP

Os registros do módulo `logging` são exportados em lotes via OTLP HTTP/protobuf, preservando os handlers do console, nível e exceções. Configure `OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector.observability.svc.cluster.local:4318`; o exporter acrescenta `/v1/logs`. `OTEL_EXPORTER_OTLP_LOGS_ENDPOINT` permite uma URL completa específica. `OTEL_LOGS_EXPORTER=none` desativa a exportação. `print()` e saídas arbitrárias de subprocessos não passam pelo módulo `logging`.


## Traces OpenTelemetry

O SDK exporta spans em lotes por OTLP HTTP/protobuf ao Collector configurado em
`OTEL_EXPORTER_OTLP_ENDPOINT` (acrescenta `/v1/traces`). O Collector central autentica
no New Relic EU; a aplicação não precisa da license key. Logs e métricas mantêm
suas configurações existentes. `OTEL_TRACES_EXPORTER=none` desativa os traces;
`OTEL_TRACES_SAMPLER=parentbased_traceidratio` e `OTEL_TRACES_SAMPLER_ARG=1.0`
respeitam o sampling recebido e coletam todos os traces raiz.

Flask extrai W3C `traceparent`/`tracestate`. Flag e targeting instrumentam requests
(inclusive a validação no auth-service) e consultas psycopg2, sem capturar os
parâmetros SQL. Analytics instrumenta chamadas botocore (SQS e DynamoDB), solicita
os atributos W3C da mensagem e abre um span de consumo com o contexto enviado pelo
evaluation-service. Falhas de processamento marcam o span como erro e preservam
a mensagem para retry. Bodies e headers de autenticação não são adicionados como
atributos pela aplicação. Use o processo normal de inicialização por worker;
não use Gunicorn `--preload`, pois os providers são criados na importação.
O SDK registra o shutdown via atexit; SIGTERM no entrypoint Python permite o flush.

Valide com tráfego nas rotas e, no analytics, uma mensagem SQS válida:

```sql
FROM Span SELECT count(*) FACET service.name, name SINCE 15 minutes ago
```
