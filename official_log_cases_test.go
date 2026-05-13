package grok

import "testing"

type officialLogCase struct {
	name     string
	source   string
	patterns map[string]string
	pattern  string
	line     string
}

type officialLogFixture struct {
	name       string
	source     string
	line       string
	current    *GrokRegexp
	regexpOnly *GrokRegexp
}

func loadOfficialLogCases() []officialLogCase {
	return []officialLogCase{
		{
			name:    "apache_combined_log",
			source:  "https://httpd.apache.org/docs/current/logs.html",
			pattern: `%{COMBINEDAPACHELOG}`,
			line:    `127.0.0.1 - frank [10/Oct/2000:13:55:36 -0700] "GET /apache_pb.gif HTTP/1.0" 200 2326 "http://www.example.com/start.html" "Mozilla/4.08 [en] (Win98; I ;Nav)"`,
		},
		{
			name:    "nginx_combined_access",
			source:  "https://docs.nginx.com/nginx/admin-guide/monitoring/logging/",
			pattern: `%{IPORHOST:remote_addr} - %{DATA:remote_user} \[%{HTTPDATE:time_local}\] "%{WORD:method} %{URIPATHPARAM:request} HTTP/%{NUMBER:http_version}" %{INT:status} %{INT:body_bytes_sent} "%{DATA:http_referer}" "%{DATA:http_user_agent}"`,
			line:    `192.0.2.10 - - [15/Mar/2024:14:26:02 +0000] "GET /en-US/docs/?q=grok HTTP/1.1" 200 1024 "https://example.com/start.html" "Mozilla/5.0"`,
		},
		{
			name:   "nginx_error_upstream",
			source: "https://docs.nginx.com/nginx/admin-guide/monitoring/logging/",
			patterns: map[string]string{
				"date2": `%{YEAR}[./]%{MONTHNUM}[./]%{MONTHDAY} %{TIME}`,
			},
			pattern: `%{date2:time} \[%{LOGLEVEL:status}\] %{GREEDYDATA:msg}, client: %{NOTSPACE:client_ip}, server: %{NOTSPACE:server}, request: "%{DATA:http_method} %{GREEDYDATA:http_url} HTTP/%{NUMBER:http_version}", (upstream: "%{GREEDYDATA:upstream}", )?host: "%{NOTSPACE:ip_or_host}"`,
			line:    `2024/03/15 14:26:02 [error] upstream timed out while reading response header from upstream, client: 192.0.2.10, server: api.example.com, request: "GET /api/orders HTTP/1.1", upstream: "http://10.0.0.9:8080/api/orders", host: "api.example.com"`,
		},
		{
			name:    "postgresql_duration_statement",
			source:  "https://www.postgresql.org/docs/current/runtime-config-logging.html",
			pattern: `%{TIMESTAMP_ISO8601:time} \[%{POSINT:pid}\] %{USER:user}@%{WORD:database} %{WORD:severity}:  duration: %{NUMBER:duration_ms} ms  statement: %{GREEDYDATA:statement}`,
			line:    `2024-03-15 14:26:02.123 [12345] app@appdb LOG:  duration: 12.345 ms  statement: SELECT 1`,
		},
		{
			name:   "kafka_server_log",
			source: "https://kafka.apache.org/documentation/",
			patterns: map[string]string{
				"date1": `%{INT}-%{INT}-%{INT} %{INT}:%{INT}:%{INT},%{INT}`,
			},
			pattern: `^\[%{date1:time}\] %{WORD:status} %{DATA:msg} \(%{DATA:name}\)`,
			line:    `[2024-03-15 14:26:02,123] INFO [GroupCoordinator 1]: Stabilized group order-consumer generation 42 (__consumer_offsets-12) (kafka.coordinator.group.GroupCoordinator)`,
		},
		{
			name:    "rabbitmq_default_log",
			source:  "https://www.rabbitmq.com/docs/logging",
			pattern: `%{DATA:time} \[%{LOGLEVEL:status}\] %{GREEDYDATA:msg}`,
			line:    `2024-03-15 14:26:02.123456+00:00 [info] <0.223.0> accepting AMQP connection 192.0.2.10:54321 -> 192.0.2.20:5672`,
		},
		{
			name:   "redis_server_log",
			source: "https://redis.io/docs/latest/operate/rs/7.8/clusters/logging/rsyslog-logging/",
			patterns: map[string]string{
				"date2": `%{MONTHDAY} %{MONTH} %{YEAR}?%{TIME}`,
			},
			pattern: `%{INT:pid}:%{WORD:role} %{date2:time} %{NOTSPACE:severity} %{GREEDYDATA:msg}`,
			line:    `28550:M 08 May 2024 20:20:13.012 * Ready to accept connections`,
		},
		{
			name:    "tomcat_access_log",
			source:  "https://tomcat.apache.org/tomcat-8.5-doc/config/valve.html",
			pattern: `%{NOTSPACE:client_ip} %{NOTSPACE:http_ident} %{NOTSPACE:http_auth} \[%{HTTPDATE:time}\] "%{DATA:http_method} %{GREEDYDATA:http_url} HTTP/%{NUMBER:http_version}" %{INT:status_code} %{INT:bytes}`,
			line:    `127.0.0.1 - - [15/Mar/2024:14:26:02 +0000] "GET /manager/html HTTP/1.1" 200 4312`,
		},
		{
			name:    "elasticsearch_search_slow_log",
			source:  "https://www.elastic.co/docs/reference/elasticsearch/index-settings/slow-log",
			pattern: `^\[%{TIMESTAMP_ISO8601:time}\]\[%{LOGLEVEL:status}%{SPACE}\]\[i.s.s.(?:query|fetch)%{SPACE}\] (?:\[%{HOSTNAME:nodeId}\] )?\[%{NOTSPACE:index}\]\[%{INT}\] took\[.*\], took_millis\[%{INT:duration}\].*`,
			line:    `[2024-03-15T14:26:02,123][WARN ][i.s.s.query] [node-1] [products][0] took[120ms], took_millis[120], types[], stats[], search_type[QUERY_THEN_FETCH], total_shards[5], source[{}], id[]`,
		},
		{
			name:   "solr_request_log",
			source: "https://solr.apache.org/guide/",
			patterns: map[string]string{
				"solrReporter": `(?:[.\w\d]+)`,
				"solrParams":   `(?:[A-Za-z0-9$.+!*'|(){},~@#%&/=:;_?\-\[\]<>]*)`,
				"solrPath":     `(?:%{PATH}|null)`,
			},
			pattern: `%{TIMESTAMP_ISO8601:time}%{SPACE}%{LOGLEVEL:status}%{SPACE}\(%{NOTSPACE:thread}\)%{SPACE}\[%{SPACE}%{NOTSPACE}?\]%{SPACE}%{solrReporter:reporter}%{SPACE}\[%{NOTSPACE:core}\]%{SPACE}webapp=%{NOTSPACE:webapp}%{SPACE}path=%{solrPath:path}%{SPACE}params=\{%{solrParams:params}\}(?:%{SPACE}hits=%{NUMBER:hits})?%{SPACE}status=%{NUMBER:qstatus}%{SPACE}QTime=%{NUMBER:qtime}`,
			line:    `2013-10-01 12:33:08.319 INFO (qtp12345-17) [   ] org.apache.solr.core.SolrCore [collection1] webapp=/solr path=/select params={q=*:*} hits=10 status=0 QTime=3`,
		},
	}
}

func loadOfficialLogFixtures(t testing.TB) []officialLogFixture {
	t.Helper()

	cases := loadOfficialLogCases()
	fixtures := make([]officialLogFixture, 0, len(cases))
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

		fixtures = append(fixtures, officialLogFixture{
			name:       tc.name,
			source:     tc.source,
			line:       tc.line,
			current:    current,
			regexpOnly: regexpOnly,
		})
	}
	return fixtures
}

func TestOfficialLogCasesMatchRegexp(t *testing.T) {
	for _, fixture := range loadOfficialLogFixtures(t) {
		t.Run(fixture.name, func(t *testing.T) {
			assertRunParity(t, fixture.current, fixture.regexpOnly, fixture.line, true)
			if _, err := fixture.current.Run(fixture.line, true); err != nil {
				t.Fatalf("%s fixture from %s did not match: %v", fixture.name, fixture.source, err)
			}
		})
	}
}

func TestOfficialLogCasesWithTypeInfoMatchRegexp(t *testing.T) {
	for _, fixture := range loadOfficialLogFixtures(t) {
		t.Run(fixture.name, func(t *testing.T) {
			assertTypedRunParity(t, fixture.current, fixture.regexpOnly, fixture.line, true)
		})
	}
}

func BenchmarkOfficialLogCases(b *testing.B) {
	for _, fixture := range loadOfficialLogFixtures(b) {
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

func BenchmarkOfficialLogCasesWithTypeInfo(b *testing.B) {
	for _, fixture := range loadOfficialLogFixtures(b) {
		fixture := fixture
		b.Run(fixture.name+"/fast", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				ret, err := fixture.current.RunWithTypeInfo(fixture.line, true)
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
				ret, err := fixture.regexpOnly.RunWithTypeInfo(fixture.line, true)
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
