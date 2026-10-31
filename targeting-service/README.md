# targeting-service (Python)

Este é o serviço de regras de segmentação (targeting) do projeto ToggleMaster. Ele é responsável por gerenciar regras complexas (ex: "50% dos usuários", "usuários do país X") para uma feature flag específica.

**IMPORTANTE:** Este serviço também é protegido e depende que o `auth-service` esteja rodando (ex: em `http://localhost:8001`).

## 📦 Pré-requisitos (Local)

* [Python](https://www.python.org/) (versão 3.9 ou superior)
* [PostgreSQL](https://www.postgresql.org/download/)
* O `auth-service` deve estar rodando.

## 🚀 Rodando Localmente

1.  **Clone o repositório** e entre na pasta `targeting-service`.

2.  **Prepare o Banco de Dados:**
    * Crie um banco de dados no seu PostgreSQL (ex: `targeting_db`).
    * Execute o script `db/init.sql` para criar a tabela `targeting_rules`:
        ```bash
        psql -U seu_usuario -d targeting_db -f db/init.sql
        ```

3.  **Configure as Variáveis de Ambiente:**
    Crie um arquivo chamado `.env` na raiz desta pasta (`targeting-service/`) com o seguinte conteúdo:
    ```.env
    # String de conexão do seu banco de dados PostgreSQL
    DATABASE_URL="postgres://SEU_USUARIO:SUA_SENHA@localhost:5432/targeting_db"
    
    # Porta que este serviço (targeting-service) irá rodar
    PORT="8003"
    
    # URL do auth-service (que deve estar rodando na porta 8001)
    AUTH_SERVICE_URL="http://localhost:8001"
    ```

4.  **Instale as Dependências:**
    ```bash
    pip install -r requirements.txt
    ```

5.  **Inicie o Serviço:**
    ```bash
    gunicorn --bind 0.0.0.0:8003 app:app
    ```
    O servidor estará rodando em `http://localhost:8003`.

## 🧪 Testando os Endpoints

Lembre-se de obter sua `SUA_CHAVE_API` no `auth-service` (veja o README do `flag-service`).

**1. Verifique a Saúde (Health Check):**
```bash
curl http://localhost:8003/health
```
Saída esperada: `{"status":"ok"}`

**2. Crie uma nova Regra de Segmentação:** Vamos criar uma regra para a flag enable-new-dashboard (que você criou no flag-service). Esta regra fará a flag aparecer para 50% dos usuários.
```bash
curl -X POST http://localhost:8003/rules \
-H "Content-Type: application/json" \
-H "Authorization: Bearer SUA_CHAVE_API" \
-d '{
    "flag_name": "enable-new-dashboard",
    "is_enabled": true,
    "rules": {
        "type": "PERCENTAGE",
        "value": 50
    }
}'
```
Saída esperada: (Um JSON com os dados da regra criada).

**3. Busque a Regra que você criou:**
```bash
curl http://localhost:8003/rules/enable-new-dashboard \
-H "Authorization: Bearer SUA_CHAVE_API"
```
Saída esperada: (O JSON da regra que você acabou de criar).

**4. Atualize a Regra (mude para 75%):**
```bash
curl -X PUT http://localhost:8003/rules/enable-new-dashboard \
-H "Content-Type: application/json" \
-H "Authorization: Bearer SUA_CHAVE_API" \
-d '{
    "rules": {
        "type": "PERCENTAGE",
        "value": 75
    }
}'
```
Saída esperada: (O JSON da regra atualizada, com `"value": 75`).

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
