package grok

import (
	"reflect"
	"testing"
)

type realisticPatternCase struct {
	name    string
	pattern string
	line    string
}

type realisticPatternFixture struct {
	name       string
	line       string
	current    *GrokRegexp
	regexpOnly *GrokRegexp
}

func loadRealisticPatternCases() []realisticPatternCase {
	return []realisticPatternCase{
		{
			name:    "business_order_logfmt",
			pattern: `time=%{TIMESTAMP_ISO8601:time} level=%{LOGLEVEL:level} service=%{NOTSPACE:service} trace_id=%{NOTSPACE:trace_id} user_id=%{INT:user_id} order_id=%{NOTSPACE:order_id} amount=%{NUMBER:amount} latency_ms=%{NUMBER:latency_ms} msg="%{GREEDYDATA:msg}"`,
			line:    `time=2026-04-24T10:31:42.123+08:00 level=INFO service=order-api trace_id=4f8c9c1b0d9a user_id=10086 order_id=ORD-20260424-00042 amount=319.95 latency_ms=12.7 msg="created order from checkout"`,
		},
		{
			name:    "business_payment_worker",
			pattern: `%{TIMESTAMP_ISO8601:time} %{LOGLEVEL:level} worker=%{NOTSPACE:worker} tenant=%{NOTSPACE:tenant} order_id=%{NOTSPACE:order_id} payment_id=%{NOTSPACE:payment_id} amount=%{NUMBER:amount} currency=%{WORD:currency} status=%{WORD:status} cost=%{NUMBER:cost_ms}ms`,
			line:    `2026-04-24T10:31:42.456+08:00 INFO worker=payment-settle-3 tenant=shop-cn order_id=ORD-20260424-00042 payment_id=PAY-99a01 amount=319.95 currency=CNY status=confirmed cost=8.4ms`,
		},
		{
			name:    "api_gateway_access",
			pattern: `%{IPORHOST:client} %{WORD:method} %{URIPATHPARAM:path} status=%{INT:status} bytes=%{INT:bytes} duration=%{NUMBER:duration_ms} trace=%{NOTSPACE:trace_id}`,
			line:    `10.20.30.40 POST /api/v1/orders?channel=app status=201 bytes=532 duration=12.4 trace=4f8c9c1b0d9a`,
		},
		{
			name:    "java_order_service",
			pattern: `%{TIMESTAMP_ISO8601:time} \[%{LOGLEVEL:level}\] \[%{NOTSPACE:thread}\] %{NOTSPACE:logger} - order_id=%{NOTSPACE:order_id} user_id=%{INT:user_id} %{GREEDYDATA:msg}`,
			line:    `2026-04-24 10:31:42,789 [WARN] [http-nio-8080-exec-7] com.demo.OrderService - order_id=ORD-20260424-00042 user_id=10086 inventory reservation timeout after 200ms`,
		},
		{
			name:    "python_gunicorn_order_access",
			pattern: `%{IPORHOST:client} - - \[%{HTTPDATE:time}\] "%{WORD:method} %{URIPATHPARAM:path} HTTP/%{NUMBER:http_version}" %{INT:status} %{INT:bytes} "%{GREEDYDATA:referrer}" "%{GREEDYDATA:agent}"`,
			line:    `10.20.30.40 - - [24/Apr/2026:10:31:42 +0800] "POST /api/v1/orders?channel=app HTTP/1.1" 201 532 "-" "Mozilla/5.0 checkout-app/8.6.1"`,
		},
		{
			name:    "k8s_controller_runtime",
			pattern: `%{TIMESTAMP_ISO8601:time}%{SPACE}%{LOGLEVEL:level}%{SPACE}%{NOTSPACE:logger}%{SPACE}%{DATA:msg}%{SPACE}controller=%{NOTSPACE:controller}%{SPACE}resource=%{GREEDYDATA:resource}%{SPACE}name=%{NOTSPACE:name}%{SPACE}namespace=%{NOTSPACE:namespace}`,
			line:    `2026-04-24T10:31:42Z INFO controller-runtime.manager reconciled object controller=orders resource=apps/v1, Kind=Deployment name=order-api namespace=prod`,
		},
		{
			name:    "syslog_auth_failure",
			pattern: `%{SYSLOGTIMESTAMP:timestamp} %{SYSLOGHOST:host} %{WORD:program}(?:\[%{POSINT:pid}\])?: %{GREEDYDATA:msg}`,
			line:    `Apr 24 10:31:42 edge-01 sshd[24731]: Failed password for invalid user admin from 203.0.113.9 port 54022 ssh2`,
		},
		{
			name:    "db_slow_query",
			pattern: `%{TIMESTAMP_ISO8601:time}%{SPACE}%{INT:thread_id}%{SPACE}%{WORD:operation}%{SPACE}%{GREEDYDATA:raw_query}`,
			line:    `2026-04-24T10:31:42.981+08:00 183742 SELECT SELECT * FROM orders WHERE user_id = 10086 AND status = 'pending' ORDER BY created_at DESC LIMIT 20`,
		},
		{
			name:    "optional_trace_worker",
			pattern: `%{TIMESTAMP_ISO8601:time} %{LOGLEVEL:level} worker=%{NOTSPACE:worker} (trace=%{NOTSPACE:trace_id} )?msg=%{GREEDYDATA:msg}`,
			line:    `2026-04-24T10:31:42+08:00 INFO worker=inventory-sync-1 trace=4f8c9c1b0d9a msg=reservation refreshed successfully`,
		},
		{
			name:    "message_queue_consumer",
			pattern: `%{TIMESTAMP_ISO8601:time} level=%{LOGLEVEL:level} topic=%{NOTSPACE:topic} partition=%{INT:partition} offset=%{INT:offset} group=%{NOTSPACE:group} lag=%{INT:lag} msg="%{GREEDYDATA:msg}"`,
			line:    `2026-04-24T10:31:42.222Z level=INFO topic=order-events partition=12 offset=992331 group=checkout-consumer lag=3 msg="processed order created event"`,
		},
	}
}

func loadRealisticPatternFixtures(t testing.TB) []realisticPatternFixture {
	t.Helper()

	denorm, errs := DenormalizePatternsFromMap(CopyDefalutPatterns())
	if len(errs) != 0 {
		t.Fatalf("denormalize default patterns: %v", errs)
	}

	cases := loadRealisticPatternCases()
	fixtures := make([]realisticPatternFixture, 0, len(cases))
	for _, tc := range cases {
		current, err := CompilePattern(tc.pattern, PatternStorage{denorm})
		if err != nil {
			t.Fatalf("%s compile current: %v", tc.name, err)
		}
		regexpOnly, err := CompilePattern(tc.pattern, PatternStorage{denorm})
		if err != nil {
			t.Fatalf("%s compile regexp: %v", tc.name, err)
		}
		regexpOnly.fastMatcher = nil
		fixtures = append(fixtures, realisticPatternFixture{
			name:       tc.name,
			line:       tc.line,
			current:    current,
			regexpOnly: regexpOnly,
		})
	}
	return fixtures
}

func TestRealisticPatternsMatchRegexp(t *testing.T) {
	for _, fixture := range loadRealisticPatternFixtures(t) {
		fastRet, fastErr := fixture.current.Run(fixture.line, true)
		regexpRet, regexpErr := fixture.regexpOnly.Run(fixture.line, true)
		if fastErr != regexpErr || !reflect.DeepEqual(fastRet, regexpRet) {
			t.Fatalf("%s diverged: fast=%v/%v regexp=%v/%v", fixture.name, fastRet, fastErr, regexpRet, regexpErr)
		}
	}
}

func TestRealisticPatternsMutationParity(t *testing.T) {
	extraLines := map[string][]string{
		"business_order_logfmt": {
			`time=2026-04-24T10:31:42.123+08:00 level=INFO service=order-api trace_id=4f8c9c1b0d9a user_id=10086 order_id=ORD-20260424-00042 amount=+.95 latency_ms=12.7 msg="created order from checkout"`,
			`time=2026-04-24T10:31:42.123+08:00 level=INFO service=order-api trace_id=4f8c9c1b0d9a user_id=10086 order_id=ORD-20260424-00042 amount=319.95 latency_ms= msg="created order from checkout"`,
		},
		"business_payment_worker": {
			`2026-04-24T10:31:42.456+08:00 INFO worker=payment-settle-3 tenant=shop-cn order_id=ORD-20260424-00042 payment_id=PAY-99a01 amount=.95 currency=CNY status=confirmed cost=8.4ms`,
			`2026-04-24T10:31:42.456+08:00 INFO worker=payment-settle-3 tenant=shop-cn order_id=ORD-20260424-00042 payment_id=PAY-99a01 amount=319.95 currency=CNY status=confirmed cost=+.4ms`,
		},
		"api_gateway_access": {
			`prefix 10.20.30.40 POST /api/v1/orders?channel=app status=201 bytes=532 duration=12.4 trace=4f8c9c1b0d9a`,
			`10.20.30.40 POST /api/v1/orders?channel=app status=201 bytes=532 duration=+.4 trace=4f8c9c1b0d9a`,
		},
		"java_order_service": {
			`2026-04-24 10:31:42,789 [WARN] [http-nio-8080-exec-7] com.demo.OrderService - order_id=ORD-20260424-00042 user_id=10086`,
			`2026-04-24 10:31:42,789 [WARN] [] com.demo.OrderService - order_id=ORD-20260424-00042 user_id=10086 inventory reservation timeout after 200ms`,
		},
		"python_gunicorn_order_access": {
			`prefix 10.20.30.40 - - [24/Apr/2026:10:31:42 +0800] "POST /api/v1/orders?channel=app HTTP/1.1" 201 532 "-" "Mozilla/5.0 checkout-app/8.6.1"`,
			`10.20.30.40  - [24/Apr/2026:10:31:42 +0800] "POST /api/v1/orders?channel=app HTTP/1.1" 201 532 "-" "Mozilla/5.0 checkout-app/8.6.1"`,
			`10.20.30.40 - - [24/Apr/2026:10:31:42 +0800] "POST /api/v1/orders?channel=app HTTP/+.1" 201 532 "-" "Mozilla/5.0 checkout-app/8.6.1"`,
		},
		"k8s_controller_runtime": {
			`2026-04-24T10:31:42Z INFO controller-runtime.manager reconciled object controller=orders resource=apps/v1, Kind=Deployment name=order-api`,
			`2026-04-24T10:31:42Z INFO controller-runtime.manager  controller=orders resource=apps/v1, Kind=Deployment name=order-api namespace=prod`,
		},
		"syslog_auth_failure": {
			`Apr 24 10:31:42 edge-01 sshd: Failed password for invalid user admin from 203.0.113.9 port 54022 ssh2`,
			`Apr 24 10:31:42 edge-01 sshd[]: Failed password for invalid user admin from 203.0.113.9 port 54022 ssh2`,
		},
		"db_slow_query": {
			`2026-04-24T10:31:42.981+08:00 183742 SELECT`,
			`2026-04-24T10:31:42.981+08:00 x SELECT SELECT * FROM orders`,
		},
		"optional_trace_worker": {
			`2026-04-24T10:31:42+08:00 INFO worker=inventory-sync-1 msg=reservation refreshed successfully`,
			`2026-04-24T10:31:42+08:00 INFO worker=inventory-sync-1 trace= msg=reservation refreshed successfully`,
		},
		"message_queue_consumer": {
			`2026-04-24T10:31:42.222Z level=INFO topic=order-events partition=12 offset=992331 group=checkout-consumer lag=3 msg=""`,
			`2026-04-24T10:31:42.222Z level=INFO topic=order-events partition=12 offset=992331 group=checkout-consumer lag=x msg="processed order created event"`,
		},
	}

	for _, fixture := range loadRealisticPatternFixtures(t) {
		lines := append([]string{fixture.line, fixture.line + "\n"}, extraLines[fixture.name]...)
		for idx, line := range lines {
			t.Run(fixture.name, func(t *testing.T) {
				assertRunParity(t, fixture.current, fixture.regexpOnly, line, idx != 0)
			})
		}
	}
}

func BenchmarkRealisticPatterns(b *testing.B) {
	fixtures := loadRealisticPatternFixtures(b)
	if len(fixtures) == 0 {
		b.Fatal("expected realistic fixtures")
	}

	for _, fixture := range fixtures {
		fixture := fixture
		b.Run(fixture.name+"/fast", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				ret, err := fixture.current.Run(fixture.line, true)
				if err != nil {
					b.Fatal(err)
				}
				if len(ret) == 0 {
					b.Fatal("empty result")
				}
			}
		})
		b.Run(fixture.name+"/fast_reuse", func(b *testing.B) {
			buf := make([]string, 0, fixture.current.MatchCount())
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				ret, err := fixture.current.RunTo(fixture.line, true, buf)
				if err != nil {
					b.Fatal(err)
				}
				if len(ret) == 0 {
					b.Fatal("empty result")
				}
			}
		})
		b.Run(fixture.name+"/regexp", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				ret, err := fixture.regexpOnly.Run(fixture.line, true)
				if err != nil {
					b.Fatal(err)
				}
				if len(ret) == 0 {
					b.Fatal("empty result")
				}
			}
		})
	}
}
