package grok

import (
	"errors"
	"reflect"
	"testing"
)

type typedMutationFixture struct {
	name       string
	current    *GrokRegexp
	regexpOnly *GrokRegexp
	lines      []string
}

func TestCommonComposedPatternsMutationParity(t *testing.T) {
	fixtures := loadCommonPatternFixtures(t)
	extraLines := map[string][]string{
		"go_logfmt_service": {
			`  time=2026-04-22T10:11:12.123+08:00 level=ERROR logger=checkout.worker msg="sync failed for order 42" err="dial tcp 10.0.0.8:443: i/o timeout"  `,
		},
		"java_logback": {
			`0000-0-0 0000:0+0 [WAR] [0] 0 - `,
		},
		"k8s_controller_runtime": {
			`00-1-1 0000INFO00 0 controller=0resource=name=0namespace=0`,
		},
		"zap_console": {
			`INFO 0000-01-01T00000:000000000000000000000000000000000000000000000000000000000000`,
		},
	}

	for _, fixture := range fixtures {
		lines := append([]string{fixture.line, fixture.line + "\n"}, extraLines[fixture.name]...)
		for idx, line := range lines {
			t.Run(fixture.name+"/mut_"+string(rune('a'+idx)), func(t *testing.T) {
				assertRunParity(t, fixture.current, fixture.regexpOnly, line, idx != 0)
			})
		}
	}
}

func TestDatakitFixturesMutationParity(t *testing.T) {
	fixtures, _ := loadMatchedDatakitFixtures(t)
	extraLines := map[string][]string{
		"apache/apache/apache/Apache_access_log": {
			` - - [1/MA/00:0:00:0 0] "0000  HTTP/0" 0 `,
		},
		"dameng/dameng/dameng/dameng_log": {
			"00-1-1 0:00:0[INFO] 0\n",
			"00-1-1 0:00:0[INFO]\v",
		},
		"elasticsearch/elasticsearch/elasticsearch/ElasticSearch_log": {
			`[00-1-1 0000][WAR][0]]0`,
		},
		"nginx/nginx/nginx/Nginx_error_log2": {
			`00.1/1 0:70:0 [emerg] `,
		},
	}

	for _, fixture := range fixtures {
		lines := append([]string{fixture.line}, extraLines[fixture.name]...)
		for idx, line := range lines {
			t.Run(fixture.name+"/mut_"+string(rune('a'+idx)), func(t *testing.T) {
				assertRunParity(t, fixture.current, fixture.regexpOnly, line, idx == 0)
			})
		}
	}
}

func TestTypedPatternsMutationParity(t *testing.T) {
	fixtures := loadTypedMutationFixtures(t)
	for _, fixture := range fixtures {
		for idx, line := range fixture.lines {
			t.Run(fixture.name+"/mut_"+string(rune('a'+idx)), func(t *testing.T) {
				assertTypedRunParity(t, fixture.current, fixture.regexpOnly, line, idx == 0)
			})
		}
	}
}

func TestAccessPatternsSearchAndTypedParity(t *testing.T) {
	cases := []struct {
		name    string
		pattern string
		lines   []string
	}{
		{
			name:    "nginx_access_unanchored",
			pattern: `%{NOTSPACE:client_ip:string} %{NOTSPACE:http_ident:string} %{NOTSPACE:http_auth:string} \[%{HTTPDATE:time:string}\] "%{DATA:http_method:string} %{GREEDYDATA:http_url:string} HTTP/%{NUMBER:http_version:float}" %{INT:status_code:int} %{INT:bytes:int}`,
			lines: []string{
				`127.0.0.1 - admin [23/Apr/2014:22:58:32 +0200] "GET /index.php?a=1 HTTP/1.1" 404 207`,
				`prefix 127.0.0.1 - admin [23/Apr/2014:22:58:32 +0200] "GET /index.php?a=1 HTTP/1.1" 404 207`,
				`127.0.0.1  admin [23/Apr/2014:22:58:32 +0200] "GET /index.php?a=1 HTTP/1.1" 404 207`,
				`127.0.0.1 - admin [23/Apr/2014:22:58:32 +0200] "GET /index.php?a=1 HTTP/+.1" 404 207`,
				`127.0.0.1 - admin [23/Apr/2014:22:58:32 +0200] "GET /index.php?a=1 HTTP/1.1" x 207`,
			},
		},
		{
			name:    "apache_fixture_access",
			pattern: `%{IPORHOST:client:string} - - \[%{HTTPDATE:time:string}\] "%{WORD:method:string} %{URIPATHPARAM:path:string} HTTP/%{NUMBER:http_version:float}" %{INT:status:int}`,
			lines: []string{
				`10.0.0.8 - - [22/Apr/2026:10:11:12 +0800] "GET /healthz HTTP/1.1" 200`,
				`prefix 10.0.0.8 - - [22/Apr/2026:10:11:12 +0800] "GET /healthz HTTP/1.1" 200`,
				`10.0.0.8  - [22/Apr/2026:10:11:12 +0800] "GET /healthz HTTP/1.1" 200`,
				`10.0.0.8 - - [22/Apr/2026:10:11:12 +0800] "GET /healthz HTTP/+.1" 200`,
				`10.0.0.8 - - [22/Apr/2026:10:11:12 +0800] "GET /healthz HTTP/1.1" x`,
			},
		},
	}

	for _, tc := range cases {
		current, err := CompilePattern(tc.pattern, PatternStorage{defalutDenormalizedPatterns})
		if err != nil {
			t.Fatalf("%s compile current: %v", tc.name, err)
		}
		regexpOnly, err := CompilePattern(tc.pattern, PatternStorage{defalutDenormalizedPatterns})
		if err != nil {
			t.Fatalf("%s compile regexp: %v", tc.name, err)
		}
		regexpOnly.fastMatcher = nil
		for idx, line := range tc.lines {
			t.Run(tc.name+"/mut_"+string(rune('a'+idx)), func(t *testing.T) {
				assertRunParity(t, current, regexpOnly, line, true)
				assertTypedRunParity(t, current, regexpOnly, line, true)
			})
		}
	}
}

func loadTypedMutationFixtures(t testing.TB) []typedMutationFixture {
	t.Helper()

	cases := []struct {
		name    string
		pattern string
		lines   []string
	}{
		{
			name:    "pipeline_access",
			pattern: `%{IPORHOST:client:string} - - \[%{HTTPDATE:time:string}\] "%{WORD:method:string} %{URIPATHPARAM:path:string} HTTP/%{NUMBER:http_version:float}" %{INT:status:int} %{INT:bytes:int} "%{GREEDYDATA:referrer:string}" "%{GREEDYDATA:agent:string}"`,
			lines: []string{
				`10.0.0.8 - - [22/Apr/2026:10:11:12 +0800] "GET /healthz HTTP/1.1" 200 2 "-" "curl/8.0.1"`,
				`0 - - [//000000:00:00000000] "0 0 HTTP/0" 0 0 "" ""`,
				`! - - [1/Apr/0000:0:00:0 0] "0 / HTTP/0" 0 0 "" ""`,
			},
		},
		{
			name:    "worker_status",
			pattern: `%{TIMESTAMP_ISO8601:time:string} %{LOGLEVEL:level:string} ok=%{WORD:ok:bool} retries=%{INT:retries:int} duration=%{BASE10NUM:duration:float} trace=%{NOTSPACE:trace:string}`,
			lines: []string{
				`2026-04-22T10:11:12Z INFO ok=true retries=2 duration=12.4 trace=req-42`,
				`2026-04-22T10:11:12z INFO ok=true retries=2 duration=12.4 trace=req-42`,
			},
		},
	}

	fixtures := make([]typedMutationFixture, 0, len(cases))
	for _, tc := range cases {
		current, err := CompilePattern(tc.pattern, PatternStorage{defalutDenormalizedPatterns})
		if err != nil {
			t.Fatalf("%s compile current: %v", tc.name, err)
		}
		regexpOnly, err := CompilePattern(tc.pattern, PatternStorage{defalutDenormalizedPatterns})
		if err != nil {
			t.Fatalf("%s compile regexp: %v", tc.name, err)
		}
		regexpOnly.fastMatcher = nil

		fixtures = append(fixtures, typedMutationFixture{
			name:       tc.name,
			current:    current,
			regexpOnly: regexpOnly,
			lines:      tc.lines,
		})
	}

	return fixtures
}

func assertRunParity(t testing.TB, current *GrokRegexp, regexpOnly *GrokRegexp, line string, trimSpace bool) {
	t.Helper()

	fastRet, fastErr := current.Run(line, trimSpace)
	regexpRet, regexpErr := regexpOnly.Run(line, trimSpace)
	assertParity(t, fastRet, fastErr, regexpRet, regexpErr)
}

func assertTypedRunParity(t testing.TB, current *GrokRegexp, regexpOnly *GrokRegexp, line string, trimSpace bool) {
	t.Helper()

	fastRet, fastErr := current.RunWithTypeInfo(line, trimSpace)
	regexpRet, regexpErr := regexpOnly.RunWithTypeInfo(line, trimSpace)
	assertParity(t, fastRet, fastErr, regexpRet, regexpErr)
}

func assertParity(t testing.TB, fastRet any, fastErr error, regexpRet any, regexpErr error) {
	t.Helper()

	if !sameMatchError(fastErr, regexpErr) {
		t.Fatalf("error mismatch: fast=%v regexp=%v", fastErr, regexpErr)
	}
	if fastErr != nil {
		return
	}
	if !reflect.DeepEqual(fastRet, regexpRet) {
		t.Fatalf("result mismatch: fast=%#v regexp=%#v", fastRet, regexpRet)
	}
}

func sameMatchError(a error, b error) bool {
	switch {
	case a == nil || b == nil:
		return a == b
	case errors.Is(a, ErrMismatch) || errors.Is(b, ErrMismatch):
		return errors.Is(a, ErrMismatch) && errors.Is(b, ErrMismatch)
	default:
		return a.Error() == b.Error()
	}
}
