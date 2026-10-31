# auth-service (Go)

Este é o serviço de autenticação do projeto ToggleMaster. Ele é responsável por criar e validar chaves de API.

## 📦 Pré-requisitos (Local)

* [Go](https://go.dev/doc/install) (versão 1.21 ou superior)
* [PostgreSQL](https://www.postgresql.org/download/) (rodando localmente ou em um contêiner Docker)

## 🚀 Rodando Localmente

1.  **Clone o repositório** e entre na pasta `auth-service`.

2.  **Prepare o Banco de Dados:**
    * Crie um banco de dados no seu PostgreSQL (ex: `auth_db`).
    * Execute o script `db/init.sql` para criar a tabela `api_keys`:
        ```bash
        psql -U seu_usuario -d auth_db -f db/init.sql
        ```

3.  **Configure as Variáveis de Ambiente:**
    Crie um arquivo chamado `.env` na raiz desta pasta (`auth-service/`) com o seguinte conteúdo:
    ```.env
    # String de conexão do seu banco de dados PostgreSQL
    DATABASE_URL="postgres://SEU_USUARIO:SUA_SENHA@localhost:5432/auth_db"
    
    # Porta que o serviço irá rodar
    PORT="8001"
    
    # Chave mestra para criar novas chaves de API
    MASTER_KEY="admin-secreto-123"
    ```

4.  **Instale as Dependências:**
    ```bash
    go mod tidy
    ```

5.  **Inicie o Serviço:**
    ```bash
    go run .
    ```
    O servidor estará rodando em `http://localhost:8001`.

## 🧪 Testando os Endpoints

Você pode usar `curl` ou Postman.

**1. Verifique a Saúde (Health Check):**
```bash
curl http://localhost:8001/health
```

Saída esperada: `{"status":"ok"}`

**2. Crie uma nova Chave de API (requer a MASTER_KEY):**

```bash
curl -X POST http://localhost:8001/admin/keys \
-H "Content-Type: application/json" \
-H "Authorization: Bearer admin-secreto-123" \
-d '{"name": "meu-primeiro-servico"}'
``` 

Saída esperada (A SUA CHAVE SERÁ DIFERENTE):

```json
{
  "name": "meu-primeiro-servico",
  "key": "tm_key_a1b2c3d4...",
  "message": "Guarde esta chave com segurança! Você não poderá vê-la novamente."
}
```

**3. Valide a Chave que você acabou de criar: (Substitua tm_key_... pela chave exata que você recebeu no passo anterior)**

```bash
curl http://localhost:8001/validate \
-H "Authorization: Bearer tm_key_a1b2c3d4..."
```

Saída esperada: `{"message":"Chave válida"}`

**4. Teste uma Chave Inválida:**

```bash
curl http://localhost:8001/validate \
-H "Authorization: Bearer chave-errada-123"
```

Saída esperada: `Chave de API inválida ou inativa`
## Logs via OTLP

Os logs de `log.Print*` continuam no console e também são enviados por OTLP HTTP/protobuf ao Collector. Como o logger padrão Go não possui níveis, esses registros são exportados com nível INFO. Configure `OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector.observability.svc.cluster.local:4318`; o exporter acrescenta `/v1/logs`. `OTEL_EXPORTER_OTLP_LOGS_ENDPOINT` pode definir a URL completa separadamente. `OTEL_LOGS_EXPORTER=none` desativa a exportação. Encerramentos abruptos e `log.Fatal` podem perder registros ainda na fila.


## Traces no New Relic

O serviço exporta spans HTTP via OpenTelemetry e OTLP HTTP/protobuf, em lotes, com flush de até 5 segundos no encerramento normal (SIGTERM/SIGINT). O contexto W3C (`traceparent`/`tracestate`) é extraído das requisições recebidas. No evaluation-service, também é propagado nas chamadas HTTP ao flag-service e targeting-service, inclusive nas goroutines. Esses serviços precisam de instrumentação própria para produzir seus spans internos.

Configure as variáveis de traces do `.env.example`. Em produção, injete `OTEL_EXPORTER_OTLP_TRACES_HEADERS` por Secret, nunca no código ou na imagem. Use uma **ingest license key**, não uma user API key. Para conta europeia, use `https://otlp.eu01.nr-data.net:4318/v1/traces`. O endpoint específico de traces preserva os destinos existentes de logs e métricas.

Para usar um Collector, configure `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=http://otel-collector.observability.svc.cluster.local:4318/v1/traces`, remova o header de autenticação local e configure no Collector a pipeline de traces e o exporter autenticado do New Relic. Não é necessário instalar o agente Go proprietário.

`OTEL_TRACES_SAMPLER=parentbased_traceidratio` e `OTEL_TRACES_SAMPLER_ARG=1.0` coletam todos os traces raiz e respeitam a decisão do serviço chamador. Reduza a razão para controlar volume. `OTEL_TRACES_EXPORTER=none` desativa a exportação. Não são adicionados bodies, headers de autenticação ou chaves como atributos de spans.

Após reiniciar, faça chamadas às rotas da API (para evaluation, use uma flag sem cache para testar as chamadas externas) e consulte no New Relic:

```sql
FROM Span SELECT count(*) FACET service.name SINCE 15 minutes ago
```

Abra um trace do serviço e confira os spans HTTP relacionados. Erros 401/403 de exportação indicam que a chave ou região precisa ser conferida. A continuidade através dos serviços Python depende de sua instrumentação. As consultas PostgreSQL do auth-service, os comandos Redis e o envio SQS do evaluation-service produzem spans próprios. Valores de parâmetros SQL, chaves/valores Redis e bodies SQS não são exportados. O SQS recebe `traceparent`/`tracestate` como MessageAttributes; o consumidor precisa extraí-los para continuar o trace. O envio assíncrono tem timeout de 5 segundos e é aguardado no encerramento normal. Cache miss (`redis.Nil`) e consulta sem resultado (`sql.ErrNoRows`) não são marcados como falhas de infraestrutura.

Referência: https://docs.newrelic.com/docs/opentelemetry/best-practices/opentelemetry-otlp/
