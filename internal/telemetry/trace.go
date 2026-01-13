package telemetry

import (
	"context"
	"fmt"
	"interchange/config"
	"interchange/internal/core"
	"log"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

type Trace struct {
	TracerProvider *sdktrace.TracerProvider
	ServiceName    string
}

func NewTrace(conf *config.Configuration) (*Trace, error) {
	if conf == nil || !conf.Telemetry.Trace.Enabled {
		return &Trace{TracerProvider: nil, ServiceName: ""}, nil
	}
	exporter, err := otlptracehttp.New(context.Background(),
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithEndpointURL(conf.Telemetry.Trace.EndpointUrl),
		otlptracehttp.WithRetry(otlptracehttp.RetryConfig{
			Enabled:         true,             // 是否啟用重試
			InitialInterval: 5 * time.Second,  // 初次失敗後等待多久
			MaxInterval:     10 * time.Second, // 每次加倍延遲的最大值
			MaxElapsedTime:  60 * time.Second, // 單次請求最大重試時長（超過則丟棄）
		}),
		otlptracehttp.WithTimeout(30*time.Second),
	)
	if err != nil {
		log.Fatalf("failed to create otlp exporter: %v", err)
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(conf.App.Name),
		)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)
	return &Trace{
		TracerProvider: tp,
		ServiceName:    conf.App.Name,
	}, nil
}

func (t *Trace) StartSpanForLayer(
	ctx context.Context,
	spanName core.TraceSpanName,
	opts ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	var tracer trace.Tracer
	if t.TracerProvider == nil {
		tracer = noop.NewTracerProvider().Tracer("noop")
	} else {
		tracer = t.TracerProvider.Tracer(t.ServiceName)
	}
	return tracer.Start(ctx, string(spanName), opts...)
}

// ==== Handler 與 Service 皆可使用的開 span 方法 ====

// 1) Handler 專用（自動從 gin 取父 ctx 與漂亮名稱；可選擇覆寫 name）
func (t *Trace) StartSpanFromGinAuto(c *gin.Context, name ...string) (context.Context, trace.Span) {
	n := spanNameFromGin(c)
	if len(name) > 0 && strings.TrimSpace(name[0]) != "" {
		n = name[0]
	}
	ctx := t.GetTraceContext(c)
	ctx, span := t.StartSpanForLayer(ctx, core.TraceSpanName(n))
	c.Set(core.ContextTraceKey, ctx)
	return ctx, span
}

// 2) Service/Repo 專用（自動用呼叫者方法名作為 span 名稱）
func (t *Trace) StartSpanAuto(ctx context.Context, name ...string) (context.Context, trace.Span) {

	n := prettifyFuncName(callerFuncName(4))
	if n == "" {
		n = "unknown"
	}
	if len(name) > 0 && strings.TrimSpace(name[0]) != "" {
		n = name[0]
	}
	return t.StartSpanForLayer(ctx, core.TraceSpanName(n))
}

// 3) 通用入口：同一個 API 同時支援 *gin.Context 或 context.Context
//   - handler：傳 *gin.Context
//   - service：傳 context.Context
func (t *Trace) startSpanAny(parent interface{}, name ...string) (context.Context, trace.Span) {
	switch p := parent.(type) {
	case *gin.Context:
		return t.StartSpanFromGinAuto(p, name...)
	case context.Context:
		return t.StartSpanAuto(p, name...)
	default:
		// 不認得就開一個孤立的（不建議，但避免崩潰）
		ctx := context.Background()
		n := "unknown"
		if len(name) > 0 && strings.TrimSpace(name[0]) != "" {
			n = name[0]
		}
		return t.StartSpanForLayer(ctx, core.TraceSpanName(n))
	}
}

// 統一結束 span（含錯誤標註）
func (t *Trace) EndSpan(span trace.Span, err error) {
	if span == nil {
		return
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	span.End()
}

// For 下游所有 middleware/service 使用，統一取得最新 ctx
func (t *Trace) GetTraceContext(c *gin.Context) context.Context {
	if ctx, ok := c.Get(core.ContextTraceKey); ok {
		return ctx.(context.Context)
	}
	return c.Request.Context()
}
func (t *Trace) ApplyTraceAttributes(span trace.Span, obj interface{}) {
	if span == nil || obj == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			span.RecordError(fmt.Errorf("ApplyTraceAttributes panic: %v", r))
		}
	}()
	val := reflect.ValueOf(obj)
	typ := reflect.TypeOf(obj)

	if typ.Kind() == reflect.Ptr {
		val = val.Elem()
		typ = typ.Elem()
	}

	for i := 0; i < typ.NumField(); i++ {
		fieldType := typ.Field(i)
		tag := fieldType.Tag.Get("trace")
		if tag == "" {
			continue
		}

		fieldVal := val.Field(i)
		if !fieldVal.IsValid() || !fieldVal.CanInterface() {
			continue
		}

		switch fieldVal.Kind() {
		case reflect.String:
			span.SetAttributes(attribute.String(tag, fieldVal.String()))
		case reflect.Bool:
			span.SetAttributes(attribute.Bool(tag, fieldVal.Bool()))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			span.SetAttributes(attribute.Int64(tag, fieldVal.Int()))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			span.SetAttributes(attribute.Int64(tag, int64(fieldVal.Uint())))
		case reflect.Float32, reflect.Float64:
			span.SetAttributes(attribute.Float64(tag, fieldVal.Float()))
		case reflect.Slice, reflect.Array:
			if fieldVal.Type().Elem().Kind() == reflect.String {
				var strs []string
				for j := 0; j < fieldVal.Len(); j++ {
					strs = append(strs, fieldVal.Index(j).String())
				}
				span.SetAttributes(attribute.StringSlice(tag, strs))
			}
		case reflect.Struct:
			t.ApplyTraceAttributes(span, fieldVal.Interface()) // 遞迴
		case reflect.Ptr:
			if !fieldVal.IsNil() {
				t.ApplyTraceAttributes(span, fieldVal.Interface())
			}
		case reflect.Map:
			if tag != "" && fieldVal.Type().Key().Kind() == reflect.String {
				for _, key := range fieldVal.MapKeys() {
					mapKey := key.String()
					mapVal := fieldVal.MapIndex(key)

					switch mapVal.Kind() {
					case reflect.String:
						span.SetAttributes(attribute.String(tag+"."+mapKey, mapVal.String()))
					case reflect.Int, reflect.Int64:
						span.SetAttributes(attribute.Int64(tag+"."+mapKey, mapVal.Int()))
					case reflect.Float64, reflect.Float32:
						span.SetAttributes(attribute.Float64(tag+"."+mapKey, mapVal.Float()))
					case reflect.Bool:
						span.SetAttributes(attribute.Bool(tag+"."+mapKey, mapVal.Bool()))
					default:
						// 不支援的型別略過
					}
				}
			}

		}
	}
}

func (t *Trace) WithSpan(parent interface{}, name ...string) (context.Context, trace.Span, func(error)) {
	ctx, span := t.startSpanAny(parent, name...)
	end := func(err error) {
		t.EndSpan(span, err)
	}
	return ctx, span, end
}

// ==== 共用：名稱處理 ====

func prettifyFuncName(full string) string {
	// 1) 去掉路徑
	if i := strings.LastIndex(full, "/"); i >= 0 {
		full = full[i+1:]
	}
	// 2) 去掉編譯器附加的後綴：-fm、.funcN、以及奇怪的中點（·）之後的內容
	full = strings.TrimSuffix(full, "-fm")
	if i := strings.LastIndex(full, ".func"); i >= 0 {
		full = full[:i]
	}
	if i := strings.Index(full, "·"); i >= 0 { // 某些版本/平台可能出現
		full = full[:i]
	}
	// 3) 去掉前綴到第一個點（拿到 "(*Type[Arg]).Method"）
	if i := strings.Index(full, "."); i >= 0 {
		full = full[i+1:]
	}
	// 4) 移除指標與括號
	r := strings.NewReplacer("(*", "", "(", "", ")", "")
	full = r.Replace(full)
	// 5) 移除泛型型參（保留名稱）
	if i := strings.Index(full, "["); i >= 0 {
		// 只取 '[' 前（簡單處理，已足夠命名）
		full = full[:i] + full[strings.Index(full, "]")+1:]
	}
	return full
}

func spanNameFromGin(c *gin.Context) string {
	if hn := c.HandlerName(); hn != "" {
		return prettifyFuncName(hn)
	}
	route := c.FullPath()
	if route == "" {
		route = c.Request.URL.Path
	}
	return c.Request.Method + " " + route
}

func callerFuncName(skip int) string {
	pc, _, _, ok := runtime.Caller(skip)
	if !ok {
		return ""
	}
	if fn := runtime.FuncForPC(pc); fn != nil {
		return fn.Name()
	}
	return ""
}
