package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/go-redis/redis/v8"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func dependencyRecorder(t *testing.T) (*tracetest.SpanRecorder, context.Context, trace.Span) {
	t.Helper()
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	t.Cleanup(func() { otel.SetTracerProvider(previous); _ = provider.Shutdown(context.Background()) })
	ctx, parent := provider.Tracer("test").Start(context.Background(), "request")
	return recorder, ctx, parent
}

func TestRedisSpansPreserveParentAndOmitValues(t *testing.T) {
	recorder, ctx, parent := dependencyRecorder(t)
	defer parent.End()
	hook := redisTraceHook{}
	for _, failure := range []error{nil, redis.Nil, errors.New("sensitive connection error")} {
		cmd := redis.NewStringCmd(ctx, "get", "secret-cache-key")
		cmd.SetErr(failure)
		commandCtx, err := hook.BeforeProcess(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		_ = hook.AfterProcess(commandCtx, cmd)
	}
	spans := recorder.Ended()
	if len(spans) != 3 {
		t.Fatalf("got %d spans", len(spans))
	}
	for i, span := range spans {
		if span.Parent().SpanID() != parent.SpanContext().SpanID() || span.SpanKind() != trace.SpanKindClient {
			t.Error("lost Redis parent or client kind")
		}
		for _, attr := range span.Attributes() {
			if strings.Contains(attr.Value.AsString(), "secret") {
				t.Error("Redis key leaked")
			}
		}
		want := codes.Unset
		if i == 2 {
			want = codes.Error
		}
		if span.Status().Code != want {
			t.Errorf("status = %v, want %v", span.Status().Code, want)
		}
	}
}

func TestSQSSpanAndMessagePropagation(t *testing.T) {
	recorder, ctx, parent := dependencyRecorder(t)
	defer parent.End()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			MessageAttributes map[string]struct{ StringValue string }
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Error(err)
		}
		value := input.MessageAttributes["traceparent"].StringValue
		if !strings.Contains(value, parent.SpanContext().TraceID().String()) {
			t.Error("SQS trace ID lost")
		}
		if len(strings.Split(value, "-")) != 4 {
			t.Error("missing or invalid SQS traceparent")
		}
		w.Header().Set("Content-Type", "application/x-amz-json-1.0")
		_, _ = w.Write([]byte(`{"MessageId":"test-message"}`))
	}))
	defer server.Close()
	sess := session.Must(session.NewSession(&aws.Config{Region: aws.String("us-east-1"), Endpoint: aws.String(server.URL), Credentials: credentials.NewStaticCredentials("test", "test", ""), MaxRetries: aws.Int(0), DisableComputeChecksums: aws.Bool(true)}))
	app := &App{SqsSvc: sqs.New(sess), SqsQueueURL: server.URL + "/queue"}
	app.sendEvaluationEventContext(ctx, "private-user", "private-flag", true)
	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected producer span, got %d", len(spans))
	}
	span := spans[0]
	if span.Parent().SpanID() != parent.SpanContext().SpanID() || span.SpanKind() != trace.SpanKindProducer {
		t.Error("lost SQS parent or producer kind")
	}
	if span.Status().Code == codes.Error {
		t.Errorf("unexpected send error: %v", span.Status())
	}
	for _, attr := range span.Attributes() {
		if strings.Contains(attr.Value.AsString(), "private-") {
			t.Error("SQS payload leaked")
		}
	}
}

func TestSQSSendFailureIsTraced(t *testing.T) {
	recorder, ctx, parent := dependencyRecorder(t)
	defer parent.End()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-amz-json-1.0")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"__type":"InternalError","message":"test failure"}`))
	}))
	defer server.Close()
	sess := session.Must(session.NewSession(&aws.Config{Region: aws.String("us-east-1"), Endpoint: aws.String(server.URL), Credentials: credentials.NewStaticCredentials("test", "test", ""), MaxRetries: aws.Int(0)}))
	app := &App{SqsSvc: sqs.New(sess), SqsQueueURL: server.URL + "/queue"}
	app.sendEvaluationEventContext(ctx, "user", "flag", true)
	spans := recorder.Ended()
	if len(spans) != 1 || spans[0].Status().Code != codes.Error {
		t.Fatal("SQS failure was not traced")
	}
}
