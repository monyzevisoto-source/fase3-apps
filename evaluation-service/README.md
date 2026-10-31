# evaluation-service (Go)

Este é o serviço de avaliação, o "caminho quente" (hot path) do projeto ToggleMaster. É o único endpoint que os clientes finais (ex: seu app mobile, seu site) devem chamar.

Ele é otimizado para alta velocidade e baixa latência usando **cache em Redis**.

Ele funciona da seguinte forma:
1.  Recebe uma requisição (`/evaluate?user_id=...&flag_name=...`).
2.  Busca as regras da flag no **Redis**.
3.  **Se não estiver no cache (Cache MISS):**
    * Busca a definição da flag no `flag-service`.
    * Busca a regra no `targeting-service`.
    * Salva o resultado no Redis com um TTL (Time-To-Live) curto.
4.  Executa a lógica de avaliação (ex: "o usuário está nos 50%?").
5.  Retorna `true` ou `false` para o cliente.
6.  Envia *assincronamente* um evento da decisão para uma fila **AWS SQS**.

## 📦 Pré-requisitos (Local)

* [Go](https://go.dev/doc/install) (versão 1.21 ou superior)
* [Redis](https://redis.io/docs/getting-started/installation/) (rodando localmente ou em Docker)
* Os serviços `auth-service`, `flag-service` e `targeting-service` devem estar rodando.
* **Credenciais da AWS:** Para o SQS funcionar, seu terminal deve estar autenticado na AWS (ex: via `aws configure` ou variáveis de ambiente).

## 🚀 Rodando Localmente

1.  **Clone o repositório** e entre na pasta `evaluation-service`.

2.  **Crie uma Chave de API de Serviço:**
    Este serviço precisa se autenticar no `flag-service` e no `targeting-service`. Você deve criar uma chave de API para ele usando o `auth-service` (com a `MASTER_KEY`).
    ```bash
    curl -X POST http://localhost:8001/admin/keys \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer admin-secreto-123" \
    -d '{"name": "evaluation-service-key"}'
    ```
    Guarde a chave `key` retornada (ex: `tm_key_...`). Vamos chamá-la de `SUA_CHAVE_DE_SERVICO`.

3.  **Configure as Variáveis de Ambiente:**
    Crie um arquivo chamado `.env` na raiz desta pasta com o seguinte conteúdo:
    ```.env
    # Porta que este serviço irá rodar
    PORT="8004"
    
    # URL do seu Redis local
    REDIS_URL="redis://localhost:6379"
    
    # URLs dos outros serviços
    FLAG_SERVICE_URL="http://localhost:8002"
    TARGETING_SERVICE_URL="http://localhost:8003"
    
    # Chave de API que você criou no passo 2
    SERVICE_API_KEY="SUA_CHAVE_DE_SERVICO"
    
    # --- Configuração da AWS (Obrigatório para o desafio) ---
    # Cole a URL da fila SQS que você criou no console da AWS
    AWS_SQS_URL="[https://sqs.us-east-1.amazonaws.com/123456789012/sua-fila](https://sqs.us-east-1.amazonaws.com/123456789012/sua-fila)"
    
    # Região da sua fila SQS
    AWS_REGION="us-east-1" 
    ```

4.  **Instale as Dependências:**
    ```bash
    go mod tidy
    ```

5.  **Inicie o Serviço:**
    ```bash
    go run .
    ```
    O servidor estará rodando em `http://localhost:8004`.

## 🧪 Testando os Endpoints

Para os testes, vamos assumir que você já criou:
1.  Uma flag chamada `enable-new-dashboard` no `flag-service`.
2.  Uma regra para `enable-new-dashboard` no `targeting-service` do tipo `PERCENTAGE` com valor `50`.

**1. Verifique a Saúde (Health Check):**
```bash
curl http://localhost:8004/health
```
Saída esperada: `{"status":"ok"}``

**2. Teste a Avaliação:** Tente alguns IDs de usuário diferentes. O hash determinístico fará com que alguns caiam dentro dos 50% e outros fora.

```bash
# Teste User 1
curl "http://localhost:8004/evaluate?user_id=user-123&flag_name=enable-new-dashboard"
```
Saída (exemplo): `{"flag_name":"enable-new-dashboard","user_id":"user-123","result":true}`

```bash
# Teste User 2
curl "http://localhost:8004/evaluate?user_id=user-abc&flag_name=enable-new-dashboard"
```
Saída (exemplo): `{"flag_name":"enable-new-dashboard","user_id":"user-abc","result":false}`

**3. Verifique o Cache:** Execute o mesmo comando duas vezes seguidas. Na segunda vez, você verá um log "Cache HIT" no terminal do `evaluation-service`.

**4. Verifique a Fila SQS:** Após fazer as chamadas acima, vá até o console da AWS, abra sua fila SQS e verifique se as mensagens (`EvaluationEvent`) estão chegando.


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
