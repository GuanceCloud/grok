package grok

import (
	"reflect"
	"testing"
)

type commonPatternCase struct {
	name     string
	patterns map[string]string
	pattern  string
	line     string
}

type commonPatternFixture struct {
	name       string
	pattern    string
	line       string
	current    *GrokRegexp
	regexpOnly *GrokRegexp
}

func loadCommonPatternCases() []commonPatternCase {
	return []commonPatternCase{
		{
			name:    "go_logfmt_service",
			pattern: `time=%{TIMESTAMP_ISO8601:time} level=%{LOGLEVEL:level} logger=%{NOTSPACE:logger} msg="%{GREEDYDATA:msg}" err="%{GREEDYDATA:error}"`,
			line:    `time=2026-04-22T10:11:12.123+08:00 level=ERROR logger=checkout.worker msg="sync failed for order 42" err="dial tcp 10.0.0.8:443: i/o timeout"`,
		},
		{
			name: "go_gin_access",
			patterns: map[string]string{
				"GINTIME": `%{YEAR}/%{MONTHNUM}/%{MONTHDAY} - %{TIME}`,
			},
			pattern: `\[%{GINTIME:time}\] %{INT:status} %{WORD:method} %{URIPATHPARAM:path} %{IPORHOST:client} %{NUMBER:latency}ms`,
			line:    `[2026/04/22 - 10:11:12] 200 GET /api/orders?id=42 10.0.0.8 12.4ms`,
		},
		{
			name:    "go_worker_optional_trace",
			pattern: `%{TIMESTAMP_ISO8601:time} %{LOGLEVEL:level} worker=%{NOTSPACE:worker} (trace=%{NOTSPACE:trace} )?msg=%{GREEDYDATA:msg}`,
			line:    `2026-04-22T10:11:12+08:00 INFO worker=sync-1 trace=abc-123 msg=job completed successfully`,
		},
		{
			name:    "java_logback",
			pattern: `%{TIMESTAMP_ISO8601:time} \[%{LOGLEVEL:level}\] \[%{NOTSPACE:thread}\] %{NOTSPACE:logger} - %{GREEDYDATA:msg}`,
			line:    `2026-04-22 10:11:12,123 [WARN] [http-nio-8080-exec-3] com.demo.OrderService - timeout while waiting for inventory service`,
		},
		{
			name:    "java_spring_boot",
			pattern: `%{TIMESTAMP_ISO8601:time}%{SPACE}%{LOGLEVEL:level}%{SPACE}%{INT:pid} --- \[%{NOTSPACE:thread}\] %{NOTSPACE:logger}%{SPACE}:%{SPACE}%{GREEDYDATA:msg}`,
			line:    `2026-04-22 10:11:12.123  WARN 12345 --- [nio-8080-exec-1] c.demo.OrderService : timeout calling inventory service`,
		},
		{
			name:    "python_uvicorn",
			pattern: `%{TIMESTAMP_ISO8601:time} %{LOGLEVEL:level} \[%{NOTSPACE:logger}\] %{GREEDYDATA:msg}`,
			line:    `2026-04-22 10:11:12,123 INFO [uvicorn.error] Started server process [12345]`,
		},
		{
			name:    "python_gunicorn_access",
			pattern: `%{IPORHOST:client} - - \[%{HTTPDATE:time}\] "%{WORD:method} %{URIPATHPARAM:path} HTTP/%{NUMBER:http_version}" %{INT:status} %{INT:bytes} "%{GREEDYDATA:referrer}" "%{GREEDYDATA:agent}"`,
			line:    `10.0.0.8 - - [22/Apr/2026:10:11:12 +0800] "GET /healthz HTTP/1.1" 200 2 "-" "curl/8.0.1"`,
		},
		{
			name:    "node_pino",
			pattern: `level=%{INT:level} time=%{TIMESTAMP_ISO8601:time} pid=%{INT:pid} hostname=%{NOTSPACE:hostname} msg="%{GREEDYDATA:msg}"`,
			line:    `level=30 time=2026-04-22T10:11:12.123Z pid=12345 hostname=api-01 msg="request completed"`,
		},
		{
			name:    "zap_console",
			pattern: `%{LOGLEVEL:level}%{SPACE}%{TIMESTAMP_ISO8601:time}%{SPACE}%{NOTSPACE:logger}%{SPACE}%{GREEDYDATA:msg}`,
			line:    `INFO 2026-04-22T10:11:12.123+0800 checkout.worker request finished duration=12.4ms`,
		},
		{
			name:    "logrus_text",
			pattern: `time="%{TIMESTAMP_ISO8601:time}" level=%{LOGLEVEL:level} msg="%{GREEDYDATA:msg}" component=%{NOTSPACE:component}`,
			line:    `time="2026-04-22T10:11:12+08:00" level=warning msg="retrying request" component=payment`,
		},
		{
			name:    "k8s_controller_runtime",
			pattern: `%{TIMESTAMP_ISO8601:time}%{SPACE}%{LOGLEVEL:level}%{SPACE}%{NOTSPACE:logger}%{SPACE}%{DATA:msg}%{SPACE}controller=%{NOTSPACE:controller}%{SPACE}resource=%{GREEDYDATA:resource}%{SPACE}name=%{NOTSPACE:name}%{SPACE}namespace=%{NOTSPACE:namespace}`,
			line:    `2026-04-22T10:11:12Z INFO controller-runtime.manager reconciled object controller=orders resource=apps/v1, Kind=Deployment name=api namespace=prod`,
		},
		{
			name:    "app_kv",
			pattern: `%{TIMESTAMP_ISO8601:time} level=%{LOGLEVEL:level} service=%{NOTSPACE:service} request_id=%{NOTSPACE:request_id} msg="%{GREEDYDATA:msg}"`,
			line:    `2026-04-22T10:11:12Z level=INFO service=checkout request_id=req-42 msg="order created for user 42"`,
		},
		{
			name:    "syslog_program_pid",
			pattern: `%{SYSLOGTIMESTAMP:timestamp} %{SYSLOGHOST:host} %{WORD:program}(?:\[%{POSINT:pid}\])?: %{GREEDYDATA:msg}`,
			line:    `Apr 22 10:11:12 api-01 sshd[1234]: Failed password for invalid user admin from 10.10.0.5 port 54022 ssh2`,
		},
		{
			name:    "gateway_access",
			pattern: `%{IPORHOST:client} %{WORD:method} %{URIPATHPARAM:path} status=%{INT:status} bytes=%{INT:bytes} duration=%{NUMBER:duration}`,
			line:    `10.0.0.8 GET /api/orders?id=42 status=200 bytes=532 duration=12.4`,
		},
		{
			name:    "bracket_chain_optional_node",
			pattern: `\[%{TIMESTAMP_ISO8601:time}\] \[%{LOGLEVEL:level}\] \[%{NOTSPACE:component}\] (\[%{HOSTNAME:node}\] )?\[%{NOTSPACE:tenant}\] %{GREEDYDATA:msg}`,
			line:    `[2026-04-22T10:11:12,123] [WARN ] [billing.worker] [node-a] [tenant-42] rate limiter rejected 3 requests`,
		},
		{
			name: "custom_alias_nginx_error",
			patterns: map[string]string{
				"APPDATE": `%{YEAR}[./]%{MONTHNUM}[./]%{MONTHDAY} %{TIME}`,
				"UPBLOCK": `(upstream: "%{GREEDYDATA:upstream}", )?`,
			},
			pattern: `%{APPDATE:time} \[%{LOGLEVEL:level}\] %{GREEDYDATA:msg}, client: %{IPORHOST:client}, server: %{NOTSPACE:server}, request: "%{WORD:method} %{GREEDYDATA:path} HTTP/%{NUMBER:http_version}", %{UPBLOCK}host: "%{NOTSPACE:host}"`,
			line:    `2026/04/22 10:11:12 [error] upstream timed out while reading response header from upstream, client: 10.0.0.8, server: gateway.local, request: "GET /api/orders HTTP/1.1", upstream: "http://10.0.0.9:8080/api/orders", host: "demo.local"`,
		},
		{
			name: "custom_alias_postfix",
			patterns: map[string]string{
				"QUEUEID": `[0-9A-F]{10,11}`,
			},
			pattern: `%{SYSLOGTIMESTAMP:timestamp} %{SYSLOGHOST:host} %{WORD:program}\[%{POSINT:pid}\]: %{QUEUEID:queue_id}: %{GREEDYDATA:msg}`,
			line:    `Apr 22 10:11:12 mail-01 postfix[12345]: 3F4A2BC901: to=<user@example.com>, relay=none, delay=0.22, delays=0.12/0.01/0.09/0, dsn=4.4.1, status=deferred (connect to mx.example.com[1.2.3.4]:25: Connection timed out)`,
		},
	}
}

func loadCommonPatternFixtures(t testing.TB) []commonPatternFixture {
	t.Helper()

	cases := loadCommonPatternCases()
	fixtures := make([]commonPatternFixture, 0, len(cases))
	for _, tc := range cases {
		patterns := CopyDefalutPatterns()
		for k, v := range tc.patterns {
			patterns[k] = v
		}
		denorm, errs := DenormalizePatternsFromMap(patterns)
		if len(errs) != 0 {
			t.Fatalf("%s denormalize: %v", tc.name, errs)
		}

		current, err := CompilePattern(tc.pattern, PatternStorage{denorm})
		if err != nil {
			t.Fatalf("%s compile current: %v", tc.name, err)
		}
		regexpOnly, err := CompilePattern(tc.pattern, PatternStorage{denorm})
		if err != nil {
			t.Fatalf("%s compile regexp: %v", tc.name, err)
		}
		regexpOnly.fastMatcher = nil

		fixtures = append(fixtures, commonPatternFixture{
			name:       tc.name,
			pattern:    tc.pattern,
			line:       tc.line,
			current:    current,
			regexpOnly: regexpOnly,
		})
	}
	return fixtures
}

func TestCommonComposedPatternsMatchRegexp(t *testing.T) {
	fixtures := loadCommonPatternFixtures(t)
	if len(fixtures) == 0 {
		t.Fatal("expected common pattern fixtures")
	}

	fastCount := 0
	for _, fixture := range fixtures {
		if fixture.current.fastMatcher != nil {
			fastCount++
		}
		fastRet, fastErr := fixture.current.Run(fixture.line, true)
		regexpRet, regexpErr := fixture.regexpOnly.Run(fixture.line, true)
		if !reflect.DeepEqual(fastRet, regexpRet) || fastErr != regexpErr {
			t.Fatalf("%s diverged: fast=%v/%v regexp=%v/%v", fixture.name, fastRet, fastErr, regexpRet, regexpErr)
		}
	}

	if fastCount < len(fixtures)/2 {
		t.Fatalf("expected at least half of common patterns to use fast matcher, got %d/%d", fastCount, len(fixtures))
	}
}

func TestControllerRuntimePatternUsesFastMatcher(t *testing.T) {
	fixtures := loadCommonPatternFixtures(t)
	for _, fixture := range fixtures {
		if fixture.name != "k8s_controller_runtime" {
			continue
		}
		if fixture.current.fastMatcher == nil {
			t.Fatal("expected controller-runtime pattern to use structured fast matcher")
		}
		return
	}
	t.Fatal("expected controller-runtime fixture")
}

func BenchmarkCommonComposedPatterns(b *testing.B) {
	fixtures := loadCommonPatternFixtures(b)
	if len(fixtures) == 0 {
		b.Fatal("expected common pattern fixtures")
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
