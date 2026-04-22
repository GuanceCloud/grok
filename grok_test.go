package grok

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDenormalizeGlobalPatterns(t *testing.T) {
	if denormalized, errs := DenormalizePatternsFromMap(defalutPatterns); len(errs) != 0 {
		t.Error(errs)
	} else {
		if len(defalutPatterns) != len(denormalized) {
			t.Error("len(GlobalPatterns) != len(denormalized)")
		}
		for k := range denormalized {
			if _, ok := defalutPatterns[k]; !ok {
				t.Errorf("%s not exists", k)
			}
		}
	}
}

func TestNormalizeAnonymousCaptures(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: `(DEBUG|INFO|WARN|ERROR|FATAL)`, want: `(?:DEBUG|INFO|WARN|ERROR|FATAL)`},
		{in: `([.0-9a-z]*)`, want: `(?:[.0-9a-z]*)`},
		{in: `(foo)(bar)`, want: `(?:foo)(?:bar)`},
		{in: `(foo(?:bar))(baz)`, want: `(?:foo(?:bar))(?:baz)`},
		{in: `(?:LOG|ERROR)`, want: `(?:LOG|ERROR)`},
		{in: `(?P<name>LOG|ERROR)`, want: `(?P<name>LOG|ERROR)`},
		{in: `\(`, want: `\(`},
		{in: `[()]`, want: `[()]`},
		{in: `((foo))\1`, want: `((foo))\1`},
	}

	for _, tc := range cases {
		if got := normalizeAnonymousCaptures(tc.in); got != tc.want {
			t.Fatalf("normalizeAnonymousCaptures(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestStructuredIRMetadata(t *testing.T) {
	g, err := CompilePattern(`%{WORD:name} - %{INT:id}`, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}
	if g.fastMatcher == nil {
		t.Fatal("expected structured matcher")
	}
	if got, want := g.fastMatcher.ir.minWidth, 5; got != want {
		t.Fatalf("minWidth = %d, want %d", got, want)
	}
	if g.fastMatcher.ir.firstLiteral != "" {
		t.Fatalf("unexpected firstLiteral %q", g.fastMatcher.ir.firstLiteral)
	}

	optional, err := CompilePattern(`%{WORD:name}?bar`, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}
	if optional.fastMatcher == nil {
		t.Fatal("expected optional matcher")
	}
	if optional.fastMatcher.ir.firstLiteral != "" {
		t.Fatalf("optional matcher firstLiteral = %q, want empty", optional.fastMatcher.ir.firstLiteral)
	}
	if got, want := optional.fastMatcher.ir.lastLiteral, "bar"; got != want {
		t.Fatalf("lastLiteral = %q, want %q", got, want)
	}
}

func BenchmarkFindStringSubmatch(b *testing.B) {
	re := regexp.MustCompile(`(\w+):(\w+):(\w+):(\w+):(\w+):(\w+)`)

	str := "hello:world:foo:hello:world:foo"
	b.Run("FindStringSubmatch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			match := re.FindStringSubmatch(str)
			if len(match) != 7 {
				b.Fatal(match)
			}
		}
	})

	b.Run("FindStringSubmatchIndex", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			match := re.FindStringSubmatchIndex(str)
			m := make([]string, 0, 6)
			for j := 1; j < 7; j++ {
				m = append(m, str[match[2*j]:match[2*j+1]])
			}
			_ = m
		}
	})

}

func TestParse(t *testing.T) {
	patternINT, err := DenormalizePattern(defalutPatterns["INT"])
	if err != nil {
		t.Error(err)
	}

	patterns := map[string]*GrokPattern{
		"INT": patternINT,
	}

	denormalized, errs := DenormalizePatternsFromMap(defalutPatterns, patterns)
	if len(errs) != 0 {
		t.Error(errs)
	}
	g, err := CompilePattern("%{DAY:day}", PatternStorage{denormalized})
	if err != nil {
		t.Error(err)
	}
	ret, err := g.Run("Tue qds", true)
	if err != nil {
		t.Error(err)
	}

	if v, ok := g.GetValByName("day", ret); !ok || v != "Tue" {
		t.Fatalf("day should be 'Tue' have '%s'", v)
	}
}

func TestParseFromPathPattern(t *testing.T) {
	pathPatterns, err := LoadPatternsFromPath("./patterns")
	if err != nil {
		t.Error(err)
	}
	de, errs := DenormalizePatternsFromMap(pathPatterns)
	if len(errs) != 0 {
		t.Error(errs)
	}
	g, err := CompilePattern("%{DAY:day}", PatternStorage{de})
	if err != nil {
		t.Error(err)
	}
	ret, err := g.Run("Tue qds", true)
	if err != nil {
		t.Error(err)
	}

	if v, ok := g.GetValByName("day", ret); !ok || v != "Tue" {
		t.Fatalf("day should be 'Tue' have '%s'", v)
	}
}

func TestLoadPatternsFromPathErr(t *testing.T) {
	_, err := LoadPatternsFromPath("./Lorem ipsum Minim qui in.")
	if err == nil {
		t.Fatalf("AddPatternsFromPath should returns an error when path is invalid")
	}
}

func TestRunWithTypeInfo(t *testing.T) {
	tCase := []struct {
		data string
		ptn  string
		ret  []any
	}{
		{
			data: `1
true 1.1`,
			ptn: `%{INT:A:int}
%{WORD:B:bool} %{BASE10NUM:C:float}`,
			ret: []any{
				int64(1),
				true,
				float64(1.1),
			},
		},
		{
			data: `1
true 1.1`,
			ptn: `%{INT:A:int}
%{WORD:B:bool} %{BASE10NUM:C:int}`,
			ret: []any{
				int64(1),
				true,
				int64(0),
			},
		},
		{
			data: `1 ijk123abc
true 1.1`,
			ptn: `%{INT:A:int} %{WORD:S:string}
%{WORD:B:bool} %{BASE10NUM:C:int}`,
			ret: []any{
				int64(1),
				"ijk123abc",
				true,
				int64(0),
			},
		},
		{
			data: `1
true 1.1`,
			ptn: `%{INT:A}
%{WORD:B:bool} %{BASE10NUM:C:int}`,
			ret: []any{
				"1",
				true,
				int64(0),
			},
		},
	}

	for _, item := range tCase {
		t.Run(item.ptn, func(t *testing.T) {
			g, err := CompilePattern(item.ptn, PatternStorage{defalutDenormalizedPatterns})
			if err != nil {
				t.Fatal(err)
			}
			v, err := g.RunWithTypeInfo(item.data, true)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, item.ret, v)
		})
	}
}

func TestRunWithTypeInfoStructuredFastPathMatchesRegexp(t *testing.T) {
	pattern := `time=%{TIMESTAMP_ISO8601:time} status=%{INT:status:int} duration=%{NUMBER:duration:float} ok=%{WORD:ok:bool} bytes=%{INT:bytes:int} msg="%{GREEDYDATA:msg}"`
	line := `time=2026-04-22T10:11:12.123+08:00 status=200 duration=12.4 ok=true bytes=532 msg="request completed"`

	current, err := CompilePattern(pattern, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}
	if current.fastMatcher == nil {
		t.Fatal("expected typed pattern to use structured fast matcher")
	}

	regexpOnly, err := CompilePattern(pattern, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}
	regexpOnly.fastMatcher = nil

	fastRet, fastErr := current.RunWithTypeInfo(line, true)
	regexpRet, regexpErr := regexpOnly.RunWithTypeInfo(line, true)
	assert.Equal(t, regexpErr, fastErr)
	assert.Equal(t, regexpRet, fastRet)
	assert.Equal(t, []any{"2026-04-22T10:11:12.123+08:00", int64(200), float64(12.4), true, int64(532), "request completed"}, fastRet)
}

func TestCommonApacheLogRawRequest(t *testing.T) {
	g, err := CompilePattern("%{COMMONAPACHELOG}", PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}

	line := `127.0.0.1 - - [23/Apr/2014:22:58:32 +0200] "-" 404 -`
	ret, err := g.Run(line, true)
	if err != nil {
		t.Fatal(err)
	}

	clientIP, ok := g.GetValByName("clientip", ret)
	if !ok || clientIP != "127.0.0.1" {
		t.Fatalf("unexpected clientip: %v %q", ok, clientIP)
	}

	rawRequest, ok := g.GetValByName("rawrequest", ret)
	if !ok || rawRequest != "-" {
		t.Fatalf("unexpected rawrequest: %v %q", ok, rawRequest)
	}

	if verb, _ := g.GetValByName("verb", ret); verb != "" {
		t.Fatalf("unexpected verb: %q", verb)
	}
	if bytes, _ := g.GetValByName("bytes", ret); bytes != "" {
		t.Fatalf("unexpected bytes: %q", bytes)
	}
}

func TestStructuredCompositePattern(t *testing.T) {
	pattern := `%{NOTSPACE:remote_addr} - %{NOTSPACE:remote_user} \[%{DATA:time_local}\] "%{WORD:method} %{NOTSPACE:request} HTTP/%{NUMBER:http_version}" %{INT:status} %{INT:body_bytes_sent}`
	g, err := CompilePattern(pattern, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}

	line := `127.0.0.1 - admin [23/Apr/2014:22:58:32 +0200] "GET /index.php HTTP/1.1" 404 207`
	ret, err := g.Run(line, true)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "127.0.0.1", ret[g.nameIndex["remote_addr"]])
	assert.Equal(t, "admin", ret[g.nameIndex["remote_user"]])
	assert.Equal(t, "GET", ret[g.nameIndex["method"]])
	assert.Equal(t, "/index.php", ret[g.nameIndex["request"]])
	assert.Equal(t, "1.1", ret[g.nameIndex["http_version"]])
	assert.Equal(t, "404", ret[g.nameIndex["status"]])
	assert.Equal(t, "207", ret[g.nameIndex["body_bytes_sent"]])
}

func TestStructuredSQLServerPattern(t *testing.T) {
	pattern := `%{TIMESTAMP_ISO8601:time} %{NOTSPACE:origin}\s+%{GREEDYDATA:msg}`
	g, err := CompilePattern(pattern, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}

	line := `2024-06-21 09:14:18.123+00:00 sqlservr        Error: login failed for user test`
	ret, err := g.Run(line, true)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "2024-06-21 09:14:18.123+00:00", ret[g.nameIndex["time"]])
	assert.Equal(t, "sqlservr", ret[g.nameIndex["origin"]])
	assert.Equal(t, "Error: login failed for user test", ret[g.nameIndex["msg"]])
}

func TestStructuredApacheAccessPattern(t *testing.T) {
	pattern := `%{GREEDYDATA:ip_or_host} - - \[%{HTTPDATE:time}\] "%{DATA:http_method} %{GREEDYDATA:http_url} HTTP/%{NUMBER:http_version}" %{NUMBER:http_code} `
	g, err := CompilePattern(pattern, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}

	if g.fastMatcher == nil {
		t.Fatal("expected apache access pattern to use structured fast matcher")
	}

	line := `127.0.0.1 - - [17/May/2021:14:51:09 +0800] "GET /server-status?auto HTTP/1.1" 200 `
	ret, err := g.Run(line, true)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "127.0.0.1", ret[g.nameIndex["ip_or_host"]])
	assert.Equal(t, "GET", ret[g.nameIndex["http_method"]])
	assert.Equal(t, "/server-status?auto", ret[g.nameIndex["http_url"]])
	assert.Equal(t, "1.1", ret[g.nameIndex["http_version"]])
	assert.Equal(t, "200", ret[g.nameIndex["http_code"]])
}

func TestStructuredApacheErrorFallsBackToRegexp(t *testing.T) {
	pattern := `\[%{HTTPDERROR_DATE:time}\] \[%{GREEDYDATA:type}:%{GREEDYDATA:status}\] \[pid %{GREEDYDATA:pid}:tid %{GREEDYDATA:tid}\] `
	g, err := CompilePattern(pattern, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}

	if g.fastMatcher != nil {
		t.Fatal("expected apache error pattern to fall back to regexp")
	}

	line := `[Wed Jun 02 16:32:14.123456 2021] [authz_core:error] [pid 1234:tid 140735] `
	ret, err := g.Run(line, true)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "Wed Jun 02 16:32:14.123456 2021", ret[g.nameIndex["time"]])
	assert.Equal(t, "authz_core", ret[g.nameIndex["type"]])
	assert.Equal(t, "error", ret[g.nameIndex["status"]])
	assert.Equal(t, "1234", ret[g.nameIndex["pid"]])
	assert.Equal(t, "140735", ret[g.nameIndex["tid"]])
}

func TestStructuredApacheErrorPidOnlyFallsBackToRegexp(t *testing.T) {
	pattern := `\[%{HTTPDERROR_DATE:time}\] \[%{GREEDYDATA:type}:%{GREEDYDATA:status}\] \[pid %{INT:pid}\] `
	g, err := CompilePattern(pattern, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}

	if g.fastMatcher != nil {
		t.Fatal("expected apache error pid-only pattern to fall back to regexp")
	}

	line := `[Tue May 19 18:39:45.272121 2021] [access_compat:error] [pid 9802] `
	ret, err := g.Run(line, true)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "Tue May 19 18:39:45.272121 2021", ret[g.nameIndex["time"]])
	assert.Equal(t, "access_compat", ret[g.nameIndex["type"]])
	assert.Equal(t, "error", ret[g.nameIndex["status"]])
	assert.Equal(t, "9802", ret[g.nameIndex["pid"]])
}

func TestElasticsearchSearchSlowPattern(t *testing.T) {
	pattern := `^\[%{TIMESTAMP_ISO8601:time}\]\[%{LOGLEVEL:status}%{SPACE}\]\[i.s.s.(query|fetch)%{SPACE}\] (\[%{HOSTNAME:nodeId}\] )?\[%{NOTSPACE:index}\]\[%{INT}\] took\[.*\], took_millis\[%{INT:duration}\].*`
	g, err := CompilePattern(pattern, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}

	line := `[2021-06-01T11:56:06,712][WARN ][i.s.s.query              ] [master] [shopping][0] took[36.3ms], took_millis[36], total_hits[5 hits], types[], stats[], search_type[QUERY_THEN_FETCH], total_shards[1], source[{"query":{"match":{"name":{"query":"Nariko","operator":"OR","prefix_length":0,"max_expansions":50,"fuzzy_transpositions":true,"lenient":false,"zero_terms_query":"NONE","auto_generate_synonyms_phrase_query":true,"boost":1.0}}},"sort":[{"price":{"order":"desc"}}]}], id[],`
	ret, err := g.Run(line, true)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "2021-06-01T11:56:06,712", ret[g.nameIndex["time"]])
	assert.Equal(t, "WARN", ret[g.nameIndex["status"]])
	assert.Equal(t, "master", ret[g.nameIndex["nodeId"]])
	assert.Equal(t, "shopping", ret[g.nameIndex["index"]])
	assert.Equal(t, "36", ret[g.nameIndex["duration"]])
}

func TestElasticsearchIndexSlowPattern(t *testing.T) {
	pattern := `^\[%{TIMESTAMP_ISO8601:time}\]\[%{LOGLEVEL:status}%{SPACE}\]\[i.i.s.index%{SPACE}\] (\[%{HOSTNAME:nodeId}\] )?\[%{NOTSPACE:index}/%{NOTSPACE}\] took\[.*\], took_millis\[%{INT:duration}\].*`
	g, err := CompilePattern(pattern, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}

	line := `[2021-06-01T11:56:19,084][WARN ][i.i.s.index              ] [master] [shopping/X17jbNZ4SoS65zKTU9ZAJg] took[34.1ms], took_millis[34], type[_doc], id[LgC3xXkBLT9WrDT1Dovp], routing[], source[{"price":222,"name":"hello"}]`
	ret, err := g.Run(line, true)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "2021-06-01T11:56:19,084", ret[g.nameIndex["time"]])
	assert.Equal(t, "WARN", ret[g.nameIndex["status"]])
	assert.Equal(t, "master", ret[g.nameIndex["nodeId"]])
	assert.Equal(t, "shopping", ret[g.nameIndex["index"]])
	assert.Equal(t, "34", ret[g.nameIndex["duration"]])
}

func TestElasticsearchDefaultPattern(t *testing.T) {
	pattern := `^\[%{TIMESTAMP_ISO8601:time}\]\[%{LOGLEVEL:status}%{SPACE}\]\[%{NOTSPACE:name}%{SPACE}\]%{SPACE}(\[%{HOSTNAME:nodeId}\])?.*`
	g, err := CompilePattern(pattern, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}

	line := `[2021-06-01T11:45:15,927][WARN ][o.e.c.r.a.DiskThresholdMonitor] [master] high disk watermark [90%] exceeded`
	ret, err := g.Run(line, true)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "2021-06-01T11:45:15,927", ret[g.nameIndex["time"]])
	assert.Equal(t, "WARN", ret[g.nameIndex["status"]])
	assert.Equal(t, "o.e.c.r.a.DiskThresholdMonitor", ret[g.nameIndex["name"]])
	assert.Equal(t, "master", ret[g.nameIndex["nodeId"]])
}

func TestKingbaseDefaultPattern(t *testing.T) {
	patterns := CopyDefalutPatterns()
	patterns["log_date"] = `%{YEAR}-%{MONTHNUM}-%{MONTHDAY}%{SPACE}%{HOUR}:%{MINUTE}:%{SECOND}%{SPACE}(?:CST|UTC)`
	patterns["status"] = `(LOG|ERROR|FATAL|PANIC|WARNING|NOTICE|INFO)`
	denorm, errs := DenormalizePatternsFromMap(patterns)
	if len(errs) != 0 {
		t.Fatal(errs)
	}

	g, err := CompilePattern(`%{log_date:time}%{SPACE}\[%{INT:process_id}\]%{SPACE}%{status:status}:\s+%{GREEDYDATA:msg}`, PatternStorage{denorm})
	if err != nil {
		t.Fatal(err)
	}

	line := `2025-06-17 13:07:10.952 UTC [999] ERROR:  relation "sys_stat_activity" does not exist at character 240`
	ret, err := g.Run(line, true)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "2025-06-17 13:07:10.952 UTC", ret[g.nameIndex["time"]])
	assert.Equal(t, "999", ret[g.nameIndex["process_id"]])
	assert.Equal(t, "ERROR", ret[g.nameIndex["status"]])
	assert.Equal(t, `relation "sys_stat_activity" does not exist at character 240`, ret[g.nameIndex["msg"]])
}

func TestStructuredNginxAccessPattern(t *testing.T) {
	pattern := `%{NOTSPACE:client_ip} %{NOTSPACE:http_ident} %{NOTSPACE:http_auth} \[%{HTTPDATE:time}\] "%{DATA:http_method} %{GREEDYDATA:http_url} HTTP/%{NUMBER:http_version}" %{INT:status_code} %{INT:bytes}`
	g, err := CompilePattern(pattern, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}

	line := `127.0.0.1 - admin [23/Apr/2014:22:58:32 +0200] "GET /index.php?a=1 HTTP/1.1" 404 207`
	ret, err := g.Run(line, true)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "127.0.0.1", ret[g.nameIndex["client_ip"]])
	assert.Equal(t, "-", ret[g.nameIndex["http_ident"]])
	assert.Equal(t, "admin", ret[g.nameIndex["http_auth"]])
	assert.Equal(t, "23/Apr/2014:22:58:32 +0200", ret[g.nameIndex["time"]])
	assert.Equal(t, "GET", ret[g.nameIndex["http_method"]])
	assert.Equal(t, "/index.php?a=1", ret[g.nameIndex["http_url"]])
	assert.Equal(t, "1.1", ret[g.nameIndex["http_version"]])
	assert.Equal(t, "404", ret[g.nameIndex["status_code"]])
	assert.Equal(t, "207", ret[g.nameIndex["bytes"]])
}

func TestStructuredJenkinsPatternUsesFastMatcher(t *testing.T) {
	pattern := `%{TIMESTAMP_ISO8601:time} \[id=%{GREEDYDATA:id}\]\t%{GREEDYDATA:status}\t`
	g, err := CompilePattern(pattern, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}

	if g.fastMatcher == nil {
		t.Fatal("expected jenkins pattern to use structured fast matcher")
	}

	line := `2021-05-18 03:08:58.053+0000 [id=32]	INFO	jenkins.InitReactorRunner$1#onAttained: Started all plugins`
	ret, err := g.Run(line, true)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "2021-05-18 03:08:58.053+0000", ret[g.nameIndex["time"]])
	assert.Equal(t, "32", ret[g.nameIndex["id"]])
	assert.Equal(t, "INFO", ret[g.nameIndex["status"]])
}

func TestStructuredSimpleCharClassPattern(t *testing.T) {
	patterns := CopyDefalutPatterns()
	patterns["date2"] = `%{YEAR}[./]%{MONTHNUM}[./]%{MONTHDAY} %{TIME}`
	denorm, errs := DenormalizePatternsFromMap(patterns)
	if len(errs) != 0 {
		t.Fatal(errs)
	}

	pattern := `%{date2:time} \[%{LOGLEVEL:status}\] %{GREEDYDATA:msg}`
	g, err := CompilePattern(pattern, PatternStorage{denorm})
	if err != nil {
		t.Fatal(err)
	}

	if g.fastMatcher == nil {
		t.Fatal("expected simple char class pattern to use structured fast matcher")
	}

	line := `2021/04/29 16:24:38 [emerg] unexpected ";" in /usr/local/etc/nginx/nginx.conf:23`
	ret, err := g.Run(line, true)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "2021/04/29 16:24:38", ret[g.nameIndex["time"]])
	assert.Equal(t, "emerg", ret[g.nameIndex["status"]])
	assert.Equal(t, `unexpected ";" in /usr/local/etc/nginx/nginx.conf:23`, ret[g.nameIndex["msg"]])
}

func TestStructuredOptionalRefPattern(t *testing.T) {
	g, err := CompilePattern(`prefix %{WORD:name}? suffix`, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}

	if g.fastMatcher == nil {
		t.Fatal("expected optional ref pattern to use structured fast matcher")
	}

	ret, err := g.Run(`prefix hello suffix`, true)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "hello", ret[g.nameIndex["name"]])

	ret, err = g.Run(`prefix  suffix`, true)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "", ret[g.nameIndex["name"]])
}

func TestStructuredOptionalLiteralPattern(t *testing.T) {
	g, err := CompilePattern(`\[%{HOST:host}?\]`, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}

	if g.fastMatcher == nil {
		t.Fatal("expected optional literal pattern to use structured fast matcher")
	}

	ret, err := g.Run(`[example.com]`, true)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "example.com", ret[g.nameIndex["host"]])

	ret, err = g.Run(`[]`, true)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "", ret[g.nameIndex["host"]])

	dst := make([]string, len(g.subMatchNames.name))
	if !g.fastMatcher.match(dst, `[]`, true) {
		t.Fatal("expected optional literal fast matcher to match empty host")
	}
	assert.Equal(t, "", dst[g.nameIndex["host"]])
}

func TestStructuredCharClassStarPattern(t *testing.T) {
	patterns := CopyDefalutPatterns()
	patterns["session_id"] = `([.0-9a-z]*)`
	denorm, errs := DenormalizePatternsFromMap(patterns)
	if len(errs) != 0 {
		t.Fatal(errs)
	}

	pattern := `%{session_id:session_id}:`
	g, err := CompilePattern(pattern, PatternStorage{denorm})
	if err != nil {
		t.Fatal(err)
	}

	meta, err := loadCompiledRegexpMeta(g.grokPattern.denormalized)
	if err != nil {
		t.Fatal(err)
	}
	steps, ok := compileStructuredSteps(pattern, PatternStorage{denorm}, meta, 0)
	if !ok || len(steps) == 0 {
		t.Fatal("expected char class star pattern to compile into structured steps")
	}

	ret, err := g.Run(`60b48f01.12241:`, true)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "60b48f01.12241", ret[g.nameIndex["session_id"]])
}

func TestStructuredNginxErrorPatternUsesFastMatcher(t *testing.T) {
	patterns := CopyDefalutPatterns()
	patterns["date2"] = `%{YEAR}[./]%{MONTHNUM}[./]%{MONTHDAY} %{TIME}`
	denorm, errs := DenormalizePatternsFromMap(patterns)
	if len(errs) != 0 {
		t.Fatal(errs)
	}

	pattern := `%{date2:time} \[%{LOGLEVEL:status}\] %{GREEDYDATA:msg}, client: %{NOTSPACE:client_ip}, server: %{NOTSPACE:server}, request: "%{DATA:http_method} %{GREEDYDATA:http_url} HTTP/%{NUMBER:http_version}", (upstream: "%{GREEDYDATA:upstream}", )?host: "%{NOTSPACE:ip_or_host}"`
	g, err := CompilePattern(pattern, PatternStorage{denorm})
	if err != nil {
		t.Fatal(err)
	}

	if g.fastMatcher == nil {
		t.Fatal("expected nginx error pattern to use structured fast matcher")
	}

	line := `2021/04/21 09:24:04 [alert] write() to "/var/log/nginx/access.log" failed (28: No space left on device) while logging request, client: 120.204.196.129, server: localhost, request: "GET / HTTP/1.1", host: "47.98.103.73"`
	ret, err := g.Run(line, true)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "2021/04/21 09:24:04", ret[g.nameIndex["time"]])
	assert.Equal(t, "alert", ret[g.nameIndex["status"]])
	assert.Equal(t, "120.204.196.129", ret[g.nameIndex["client_ip"]])
	assert.Equal(t, "localhost", ret[g.nameIndex["server"]])
	assert.Equal(t, "GET", ret[g.nameIndex["http_method"]])
	assert.Equal(t, "/", ret[g.nameIndex["http_url"]])
	assert.Equal(t, "1.1", ret[g.nameIndex["http_version"]])
	assert.Equal(t, "47.98.103.73", ret[g.nameIndex["ip_or_host"]])
}

func TestStructuredPostgreSQLPatternUsesFastMatcher(t *testing.T) {
	var script, line string
	for _, c := range loadDatakitFixtureCases(t) {
		if c.Collector != "postgresql" {
			continue
		}
		script = c.Pipelines["postgresql"]
		line = c.Examples["postgresql"]["PostgreSQL log"]
		break
	}
	if script == "" || line == "" {
		t.Fatal("expected postgresql fixture data")
	}

	compiled, err := compileDatakitPipeline(script)
	if err != nil {
		t.Fatal(err)
	}
	if len(compiled) == 0 {
		t.Fatal("expected compiled postgresql patterns")
	}

	g := compiled[0].current
	if g.fastMatcher == nil {
		t.Fatal("expected postgresql pattern to use structured fast matcher")
	}
	if !g.fastMatcher.backtracking {
		t.Fatal("expected postgresql pattern to use backtracking fast matcher")
	}

	ret, err := g.Run(line, true)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "2021-05-31 15:23:45.110 CST", ret[g.nameIndex["time"]])
	assert.Equal(t, "74305", ret[g.nameIndex["process_id"]])
	assert.Equal(t, "test", ret[g.nameIndex["db_name"]])
	assert.Equal(t, "pgAdmin 4 - DB:postgres", ret[g.nameIndex["application_name"]])
	assert.Equal(t, "postgres", ret[g.nameIndex["user"]])
	assert.Equal(t, "127.0.0.1", ret[g.nameIndex["remote_host"]])
	assert.Equal(t, "60b48f01.12241", ret[g.nameIndex["session_id"]])
	assert.Equal(t, "LOG", ret[g.nameIndex["status"]])
}

func TestStructuredGreedyLiteralBacktracksLikeRegexp(t *testing.T) {
	pattern := `\[%{GREEDYDATA:application}\]%{SPACE}%{USER:user}%{SPACE}\[%{HOST:remote_host}\]`

	current, err := CompilePattern(pattern, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}
	if current.fastMatcher == nil {
		t.Fatal("expected greedy literal pattern to use structured fast matcher")
	}
	if !current.fastMatcher.backtracking {
		t.Fatal("expected greedy literal pattern to require backtracking")
	}

	regexpOnly, err := CompilePattern(pattern, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}
	regexpOnly.fastMatcher = nil

	line := `[pgAdmin 4 - DB:postgres] postgres [127.0.0.1]`
	fastRet, fastErr := current.Run(line, true)
	regexpRet, regexpErr := regexpOnly.Run(line, true)
	assert.Equal(t, regexpErr, fastErr)
	assert.Equal(t, regexpRet, fastRet)
	assert.Equal(t, "pgAdmin 4 - DB:postgres", fastRet[current.nameIndex["application"]])
	assert.Equal(t, "postgres", fastRet[current.nameIndex["user"]])
	assert.Equal(t, "127.0.0.1", fastRet[current.nameIndex["remote_host"]])
}

func TestStructuredOptionalLiteralBoundaryMatchesRegexp(t *testing.T) {
	pattern := `%{DATA:msg}foo?bar`

	current, err := CompilePattern(pattern, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}
	if current.fastMatcher == nil {
		t.Fatal("expected optional literal boundary pattern to use structured fast matcher")
	}

	regexpOnly, err := CompilePattern(pattern, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}
	regexpOnly.fastMatcher = nil

	for _, line := range []string{"zzzfoobar", "zzzbar"} {
		fastRet, fastErr := current.Run(line, true)
		regexpRet, regexpErr := regexpOnly.Run(line, true)
		assert.Equalf(t, regexpErr, fastErr, "optional literal boundary error diverged for %q", line)
		assert.Equalf(t, regexpRet, fastRet, "optional literal boundary result diverged for %q", line)
	}
}

func TestStructuredParserBoundaryAcrossSpaceMatchesRegexp(t *testing.T) {
	pattern := `%{NOTSPACE:name}%{SPACE}\]`

	current, err := CompilePattern(pattern, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}
	if current.fastMatcher == nil {
		t.Fatal("expected parser boundary pattern to use structured fast matcher")
	}

	regexpOnly, err := CompilePattern(pattern, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}
	regexpOnly.fastMatcher = nil

	for _, line := range []string{"foo]", "foo ]"} {
		fastRet, fastErr := current.Run(line, true)
		regexpRet, regexpErr := regexpOnly.Run(line, true)
		assert.Equalf(t, regexpErr, fastErr, "space boundary error diverged for %q", line)
		assert.Equalf(t, regexpRet, fastRet, "space boundary result diverged for %q", line)
	}
}

func TestStructuredRedisFixtureUsesFastMatcher(t *testing.T) {
	fixtures, _ := loadMatchedDatakitFixtures(t)
	for _, fixture := range fixtures {
		if fixture.collector != "redis" {
			continue
		}
		if fixture.current.fastMatcher == nil {
			t.Fatal("expected redis fixture to use structured fast matcher")
		}

		dst := make([]string, len(fixture.current.subMatchNames.name))
		if !fixture.current.fastMatcher.match(dst, fixture.line, true) {
			t.Fatal("expected redis fixture fast matcher to succeed")
		}
		if fixture.current.fastMatcher.backtracking {
			t.Fatal("expected redis fixture fast matcher to avoid backtracking")
		}

		ret, err := fixture.current.Run(fixture.line, true)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, "122", ret[fixture.current.nameIndex["pid"]])
		assert.Equal(t, "M", ret[fixture.current.nameIndex["role"]])
		assert.Equal(t, "14 May 2019 19:11:40.164", ret[fixture.current.nameIndex["time"]])
		assert.Equal(t, "*", ret[fixture.current.nameIndex["serverity"]])
		assert.Equal(t, "Background saving terminated with success", ret[fixture.current.nameIndex["msg"]])
		return
	}

	t.Fatal("expected redis fixture")
}

func TestStructuredOptionalYearBeforeTimeMatchesRegexp(t *testing.T) {
	patterns := CopyDefalutPatterns()
	patterns["date2"] = `%{MONTHDAY} %{MONTH} %{YEAR}?%{TIME}`
	denorm, errs := DenormalizePatternsFromMap(patterns)
	if len(errs) != 0 {
		t.Fatal(errs)
	}

	current, err := CompilePattern(`%{date2:time}`, PatternStorage{denorm})
	if err != nil {
		t.Fatal(err)
	}
	if current.fastMatcher == nil {
		t.Fatal("expected date2 to use structured fast matcher")
	}
	if current.fastMatcher.backtracking {
		t.Fatal("expected optional year/time matcher to avoid backtracking")
	}

	regexpOnly, err := CompilePattern(`%{date2:time}`, PatternStorage{denorm})
	if err != nil {
		t.Fatal(err)
	}
	regexpOnly.fastMatcher = nil

	cases := []string{
		`14 May 2019 19:11:40.164`,
		`14 May 19:11:40.164`,
	}

	for _, line := range cases {
		fastRet, fastErr := current.Run(line, true)
		regexpRet, regexpErr := regexpOnly.Run(line, true)
		assert.Equalf(t, regexpErr, fastErr, "optional year/time error diverged for %q", line)
		assert.Equalf(t, regexpRet, fastRet, "optional year/time result diverged for %q", line)
	}
}

func TestStructuredSolrFixtureUsesFastMatcher(t *testing.T) {
	fixtures, _ := loadMatchedDatakitFixtures(t)
	for _, fixture := range fixtures {
		if fixture.collector != "solr" {
			continue
		}
		if fixture.current.fastMatcher == nil {
			t.Fatal("expected solr fixture to use structured fast matcher")
		}

		dst := make([]string, len(fixture.current.subMatchNames.name))
		if !fixture.current.fastMatcher.match(dst, fixture.line, true) {
			t.Fatal("expected solr fixture fast matcher to succeed")
		}

		ret, err := fixture.current.Run(fixture.line, true)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, "2013-10-01 12:33:08.319", ret[fixture.current.nameIndex["time"]])
		assert.Equal(t, "INFO", ret[fixture.current.nameIndex["status"]])
		assert.Equal(t, "org.apache.solr.core.SolrCore", ret[fixture.current.nameIndex["thread"]])
		assert.Equal(t, "webapp.reporter", ret[fixture.current.nameIndex["reporter"]])
		return
	}

	t.Fatal("expected solr fixture")
}

func TestStructuredLogLevelPattern(t *testing.T) {
	pattern := `%{TIMESTAMP_ISO8601:time} \[%{LOGLEVEL:status}\] %{GREEDYDATA:msg}`
	g, err := CompilePattern(pattern, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}

	line := `2024-06-21 09:14:18+00:00 [error] database connection failed`
	ret, err := g.Run(line, true)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "2024-06-21 09:14:18+00:00", ret[g.nameIndex["time"]])
	assert.Equal(t, "error", ret[g.nameIndex["status"]])
	assert.Equal(t, "database connection failed", ret[g.nameIndex["msg"]])
}

func TestStructuredUTFMessageTrimmedLikeRegexp(t *testing.T) {
	pattern := `%{TIMESTAMP_ISO8601:time} \[%{LOGLEVEL:status}\] %{GREEDYDATA:msg}`
	g, err := CompilePattern(pattern, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}

	line := "2024-06-21 09:14:18+00:00 [INFO] \u3000中文告警：磁盘空间不足\u3000"
	ret, err := g.Run(line, true)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "2024-06-21 09:14:18+00:00", ret[g.nameIndex["time"]])
	assert.Equal(t, "INFO", ret[g.nameIndex["status"]])
	assert.Equal(t, "中文告警：磁盘空间不足", ret[g.nameIndex["msg"]])

	regexpOnly, err := CompilePattern(pattern, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}
	regexpOnly.fastMatcher = nil
	regexpRet, err := regexpOnly.Run(line, true)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, regexpRet, ret)
}

func TestStructuredMonthMatchesDefaultRegexp(t *testing.T) {
	current, err := CompilePattern(`%{MONTH:month}`, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}
	if current.fastMatcher == nil {
		t.Fatal("expected MONTH to use structured fast matcher")
	}

	regexpOnly, err := CompilePattern(`%{MONTH:month}`, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}
	regexpOnly.fastMatcher = nil

	cases := []string{
		"Jan", "Januar",
		"Feb", "Februar",
		"Mar", "Mär", "März", "Mr", "Mrz",
		"Ma", "Mai",
		"Jun", "Juni",
		"Ot", "Oct", "Okt",
		"Dec", "Dez", "Dezember",
		"nope",
	}

	for _, line := range cases {
		fastRet, fastErr := current.Run(line, true)
		regexpRet, regexpErr := regexpOnly.Run(line, true)
		assert.Equalf(t, regexpErr, fastErr, "month match error diverged for %q", line)
		assert.Equalf(t, regexpRet, fastRet, "month match result diverged for %q", line)
	}
}

func TestStructuredDayMatchesDefaultRegexp(t *testing.T) {
	current, err := CompilePattern(`%{DAY:day}`, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}
	if current.fastMatcher == nil {
		t.Fatal("expected DAY to use structured fast matcher")
	}

	regexpOnly, err := CompilePattern(`%{DAY:day}`, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}
	regexpOnly.fastMatcher = nil

	cases := []string{
		"Mon", "Monday",
		"Tue", "Tuesday",
		"Wed", "Wednesday",
		"Thu", "Thursday",
		"Fri", "Friday",
		"Sat", "Saturday",
		"Sun", "Sunday",
		"Thur", "nope",
	}

	for _, line := range cases {
		fastRet, fastErr := current.Run(line, true)
		regexpRet, regexpErr := regexpOnly.Run(line, true)
		assert.Equalf(t, regexpErr, fastErr, "day match error diverged for %q", line)
		assert.Equalf(t, regexpRet, fastRet, "day match result diverged for %q", line)
	}
}

func TestStructuredHostNameMatchesDefaultRegexp(t *testing.T) {
	current, err := CompilePattern(`%{HOSTNAME:host}`, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}
	if current.fastMatcher == nil {
		t.Fatal("expected HOSTNAME to use structured fast matcher")
	}

	regexpOnly, err := CompilePattern(`%{HOSTNAME:host}`, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}
	regexpOnly.fastMatcher = nil

	cases := []string{
		"master",
		"node-1.example.com",
		"127.0.0.1",
		"shopping][0",
	}

	for _, line := range cases {
		fastRet, fastErr := current.Run(line, true)
		regexpRet, regexpErr := regexpOnly.Run(line, true)
		assert.Equalf(t, regexpErr, fastErr, "hostname match error diverged for %q", line)
		assert.Equalf(t, regexpRet, fastRet, "hostname match result diverged for %q", line)
	}
}

func TestStructuredLogLevelMatchesDefaultRegexp(t *testing.T) {
	current, err := CompilePattern(`%{LOGLEVEL:status}`, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}
	if current.fastMatcher == nil {
		t.Fatal("expected LOGLEVEL to use structured fast matcher")
	}

	regexpOnly, err := CompilePattern(`%{LOGLEVEL:status}`, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}
	regexpOnly.fastMatcher = nil

	cases := []string{
		"alert", "TRACE", "debug", "notice", "INFO",
		"war", "warn", "warning",
		"er", "err", "error",
		"cri", "crit", "critical",
		"fatal", "SEVERE", "emerg", "emergency",
		"verbose",
	}

	for _, line := range cases {
		fastRet, fastErr := current.Run(line, true)
		regexpRet, regexpErr := regexpOnly.Run(line, true)
		assert.Equalf(t, regexpErr, fastErr, "loglevel match error diverged for %q", line)
		assert.Equalf(t, regexpRet, fastRet, "loglevel match result diverged for %q", line)
	}
}

func TestStructuredPatternRespectsOverriddenBuiltin(t *testing.T) {
	patterns := CopyDefalutPatterns()
	patterns["NOTSPACE"] = `[A-Z]+`
	denorm, errs := DenormalizePatternsFromMap(patterns)
	if len(errs) != 0 {
		t.Fatal(errs)
	}

	g, err := CompilePattern(`%{NOTSPACE:name} %{INT:code}`, PatternStorage{denorm})
	if err != nil {
		t.Fatal(err)
	}

	ret, err := g.Run("HELLO 42", true)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "HELLO", ret[g.nameIndex["name"]])
	assert.Equal(t, "42", ret[g.nameIndex["code"]])

	if _, err := g.Run("hello 42", true); err != ErrMismatch {
		t.Fatalf("expected ErrMismatch for overridden NOTSPACE, got %v", err)
	}
}

func TestStructuredPatternRespectsOverriddenLogLevel(t *testing.T) {
	patterns := CopyDefalutPatterns()
	patterns["LOGLEVEL"] = `(?:panic|fatal)`
	denorm, errs := DenormalizePatternsFromMap(patterns)
	if len(errs) != 0 {
		t.Fatal(errs)
	}

	g, err := CompilePattern(`%{TIMESTAMP_ISO8601:time} \[%{LOGLEVEL:status}\] %{GREEDYDATA:msg}`, PatternStorage{denorm})
	if err != nil {
		t.Fatal(err)
	}

	ret, err := g.Run(`2024-06-21 09:14:18+00:00 [fatal] database connection failed`, true)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "fatal", ret[g.nameIndex["status"]])

	if _, err := g.Run(`2024-06-21 09:14:18+00:00 [error] database connection failed`, true); err != ErrMismatch {
		t.Fatalf("expected ErrMismatch for overridden LOGLEVEL, got %v", err)
	}
}

func TestRunToReuseBuffer(t *testing.T) {
	g, err := CompilePattern("%{COMMONAPACHELOG}", PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}

	buf := make([]string, 0, g.matchCount()+4)
	line := `127.0.0.1 - - [23/Apr/2014:22:58:32 +0200] "GET /index.php HTTP/1.1" 404 207`

	ret, err := g.runTo(line, true, buf)
	if err != nil {
		t.Fatal(err)
	}

	if len(ret) != g.matchCount() {
		t.Fatalf("unexpected len: %d", len(ret))
	}

	clientIP, ok := g.GetValByName("clientip", ret)
	if !ok || clientIP != "127.0.0.1" {
		t.Fatalf("unexpected clientip: %v %q", ok, clientIP)
	}

	if cap(ret) != cap(buf) {
		t.Fatalf("buffer was not reused")
	}
}

func TestRunWithTypeInfoToReuseBuffer(t *testing.T) {
	g, err := CompilePattern("%{INT:A:int} %{WORD:B:bool} %{BASE10NUM:C:float}", PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}

	buf := make([]any, 0, g.matchCount()+2)
	ret, err := g.runWithTypeInfoTo("1 true 1.1", true, buf)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, []any{int64(1), true, float64(1.1)}, ret)
	if cap(ret) != cap(buf) {
		t.Fatalf("buffer was not reused")
	}
}

func TestStringBufferPoolClearsReferencesOnPut(t *testing.T) {
	g, err := CompilePattern("%{COMMONAPACHELOG}", PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}

	pool := g.newStringBufferPool()
	line := `127.0.0.1 - - [23/Apr/2014:22:58:32 +0200] "GET /index.php HTTP/1.1" 404 207`

	buf := pool.get()
	ret, err := g.runTo(line, true, buf.Values)
	if err != nil {
		t.Fatal(err)
	}
	if ret[0] == "" {
		t.Fatal("expected populated buffer")
	}
	buf.Values = ret

	pool.put(buf)
	reused := pool.get()
	reused.Values = reused.Values[:g.matchCount()]
	for i, v := range reused.Values {
		if v != "" {
			t.Fatalf("string pool retained value at %d: %q", i, v)
		}
	}
}

func TestAnyBufferPoolClearsReferencesOnPut(t *testing.T) {
	g, err := CompilePattern("%{INT:A:int} %{WORD:B:bool} %{BASE10NUM:C:float}", PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}

	pool := g.newAnyBufferPool()

	buf := pool.get()
	ret, err := g.runWithTypeInfoTo("1 true 1.1", true, buf.Values)
	if err != nil {
		t.Fatal(err)
	}
	if ret[0] == nil {
		t.Fatal("expected populated buffer")
	}
	buf.Values = ret

	pool.put(buf)
	reused := pool.get()
	reused.Values = reused.Values[:g.matchCount()]
	for i, v := range reused.Values {
		if v != nil {
			t.Fatalf("any pool retained value at %d: %#v", i, v)
		}
	}
}

func TestStringBufferPoolRun(t *testing.T) {
	g, err := CompilePattern("%{COMMONAPACHELOG}", PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}

	pool := g.newStringBufferPool()
	buf, err := pool.run(g, `127.0.0.1 - - [23/Apr/2014:22:58:32 +0200] "GET /index.php HTTP/1.1" 404 207`, true)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.put(buf)

	clientIP, ok := g.GetValByName("clientip", buf.Values)
	if !ok || clientIP != "127.0.0.1" {
		t.Fatalf("unexpected clientip: %v %q", ok, clientIP)
	}
}

func TestAnyBufferPoolRun(t *testing.T) {
	g, err := CompilePattern("%{INT:A:int} %{WORD:B:bool} %{BASE10NUM:C:float}", PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		t.Fatal(err)
	}

	pool := g.newAnyBufferPool()
	buf, err := pool.run(g, "1 true 1.1", true)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.put(buf)

	assert.Equal(t, []any{int64(1), true, float64(1.1)}, buf.Values)
}

func BenchmarkFromMap(b *testing.B) {
	pathPatterns, err := LoadPatternsFromPath("./patterns")
	if err != nil {
		b.Error(err)
	}

	for n := 0; n < b.N; n++ {
		de, errs := DenormalizePatternsFromMap(pathPatterns)
		if len(errs) != 0 {
			b.Error(err)
			b.Error(de)
		}
	}
}

func BenchmarkCompilePatternCommonApacheLog(b *testing.B) {
	for n := 0; n < b.N; n++ {
		g, err := CompilePattern("%{COMMONAPACHELOG}", PatternStorage{defalutDenormalizedPatterns})
		if err != nil {
			b.Fatal(err)
		}
		if g == nil {
			b.Fatal("nil grok regexp")
		}
	}
}

func BenchmarkRunCommonApacheLog(b *testing.B) {
	g, err := CompilePattern("%{COMMONAPACHELOG}", PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		b.Fatal(err)
	}

	line := `127.0.0.1 - - [23/Apr/2014:22:58:32 +0200] "GET /index.php HTTP/1.1" 404 207`
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		ret, err := g.Run(line, true)
		if err != nil {
			b.Fatal(err)
		}
		if len(ret) == 0 {
			b.Fatal("empty result")
		}
	}
}

func BenchmarkRunCommonApacheLogRegexpPath(b *testing.B) {
	g, err := CompilePattern("%{COMMONAPACHELOG}", PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		b.Fatal(err)
	}
	g.fastMatcher = nil

	line := `127.0.0.1 - - [23/Apr/2014:22:58:32 +0200] "GET /index.php HTTP/1.1" 404 207`
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		ret, runErr := g.Run(line, true)
		if runErr != nil {
			b.Fatal(runErr)
		}
		if len(ret) == 0 {
			b.Fatal("empty result")
		}
	}
}

func BenchmarkCompilePatternCommonApacheLogParallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			g, err := CompilePattern("%{COMMONAPACHELOG}", PatternStorage{defalutDenormalizedPatterns})
			if err != nil {
				b.Fatal(err)
			}
			if g == nil {
				b.Fatal("nil grok regexp")
			}
		}
	})
}

func BenchmarkRunCommonApacheLogParallel(b *testing.B) {
	g, err := CompilePattern("%{COMMONAPACHELOG}", PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		b.Fatal(err)
	}

	line := `127.0.0.1 - - [23/Apr/2014:22:58:32 +0200] "GET /index.php HTTP/1.1" 404 207`
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ret, runErr := g.Run(line, true)
			if runErr != nil {
				b.Fatal(runErr)
			}
			if len(ret) == 0 {
				b.Fatal("empty result")
			}
		}
	})
}

func BenchmarkRunWithTypeInfo(b *testing.B) {
	g, err := CompilePattern("%{INT:A:int} %{WORD:B:bool} %{BASE10NUM:C:float}", PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		b.Fatal(err)
	}

	line := "1 true 1.1"
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		ret, err := g.RunWithTypeInfo(line, true)
		if err != nil {
			b.Fatal(err)
		}
		if len(ret) != 3 {
			b.Fatal(ret)
		}
	}
}

func BenchmarkRunWithTypeInfoRegexpPath(b *testing.B) {
	g, err := CompilePattern("%{INT:A:int} %{WORD:B:bool} %{BASE10NUM:C:float}", PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		b.Fatal(err)
	}
	g.fastMatcher = nil

	line := "1 true 1.1"
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		ret, err := g.RunWithTypeInfo(line, true)
		if err != nil {
			b.Fatal(err)
		}
		if len(ret) != 3 {
			b.Fatal(ret)
		}
	}
}

func BenchmarkRunWithTypeInfoStructuredCommon(b *testing.B) {
	g, err := CompilePattern(`time=%{TIMESTAMP_ISO8601:time} status=%{INT:status:int} duration=%{NUMBER:duration:float} ok=%{WORD:ok:bool} bytes=%{INT:bytes:int} msg="%{GREEDYDATA:msg}"`, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		b.Fatal(err)
	}

	line := `time=2026-04-22T10:11:12.123+08:00 status=200 duration=12.4 ok=true bytes=532 msg="request completed"`
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		ret, err := g.RunWithTypeInfo(line, true)
		if err != nil {
			b.Fatal(err)
		}
		if len(ret) != 6 {
			b.Fatal(ret)
		}
	}
}

func BenchmarkRunWithTypeInfoStructuredCommonRegexpPath(b *testing.B) {
	g, err := CompilePattern(`time=%{TIMESTAMP_ISO8601:time} status=%{INT:status:int} duration=%{NUMBER:duration:float} ok=%{WORD:ok:bool} bytes=%{INT:bytes:int} msg="%{GREEDYDATA:msg}"`, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		b.Fatal(err)
	}
	g.fastMatcher = nil

	line := `time=2026-04-22T10:11:12.123+08:00 status=200 duration=12.4 ok=true bytes=532 msg="request completed"`
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		ret, err := g.RunWithTypeInfo(line, true)
		if err != nil {
			b.Fatal(err)
		}
		if len(ret) != 6 {
			b.Fatal(ret)
		}
	}
}

func BenchmarkRunCommonApacheLogTo(b *testing.B) {
	g, err := CompilePattern("%{COMMONAPACHELOG}", PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		b.Fatal(err)
	}

	line := `127.0.0.1 - - [23/Apr/2014:22:58:32 +0200] "GET /index.php HTTP/1.1" 404 207`
	buf := make([]string, 0, g.matchCount())
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		ret, runErr := g.runTo(line, true, buf)
		if runErr != nil {
			b.Fatal(runErr)
		}
		if len(ret) == 0 {
			b.Fatal("empty result")
		}
	}
}

func BenchmarkRunCommonApacheLogToParallel(b *testing.B) {
	g, err := CompilePattern("%{COMMONAPACHELOG}", PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		b.Fatal(err)
	}

	line := `127.0.0.1 - - [23/Apr/2014:22:58:32 +0200] "GET /index.php HTTP/1.1" 404 207`
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		buf := make([]string, 0, g.matchCount())
		for pb.Next() {
			ret, runErr := g.runTo(line, true, buf)
			if runErr != nil {
				b.Fatal(runErr)
			}
			if len(ret) == 0 {
				b.Fatal("empty result")
			}
		}
	})
}

func BenchmarkRunWithTypeInfoTo(b *testing.B) {
	g, err := CompilePattern("%{INT:A:int} %{WORD:B:bool} %{BASE10NUM:C:float}", PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		b.Fatal(err)
	}

	line := "1 true 1.1"
	buf := make([]any, 0, g.matchCount())
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		ret, runErr := g.runWithTypeInfoTo(line, true, buf)
		if runErr != nil {
			b.Fatal(runErr)
		}
		if len(ret) != 3 {
			b.Fatal(ret)
		}
	}
}

func BenchmarkRunCommonApacheLogWithPoolParallel(b *testing.B) {
	g, err := CompilePattern("%{COMMONAPACHELOG}", PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		b.Fatal(err)
	}

	line := `127.0.0.1 - - [23/Apr/2014:22:58:32 +0200] "GET /index.php HTTP/1.1" 404 207`
	pool := g.newStringBufferPool()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := pool.get()
			ret, runErr := g.runTo(line, true, buf.Values)
			if runErr != nil {
				b.Fatal(runErr)
			}
			if len(ret) == 0 {
				b.Fatal("empty result")
			}
			buf.Values = ret
			pool.put(buf)
		}
	})
}

func BenchmarkRunWithTypeInfoWithPoolParallel(b *testing.B) {
	g, err := CompilePattern("%{INT:A:int} %{WORD:B:bool} %{BASE10NUM:C:float}", PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		b.Fatal(err)
	}

	pool := g.newAnyBufferPool()
	line := "1 true 1.1"
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := pool.get()
			ret, runErr := g.runWithTypeInfoTo(line, true, buf.Values)
			if runErr != nil {
				b.Fatal(runErr)
			}
			if len(ret) != 3 {
				b.Fatal(ret)
			}
			buf.Values = ret
			pool.put(buf)
		}
	})
}

func BenchmarkRunCommonApacheLogWithPoolHelperParallel(b *testing.B) {
	g, err := CompilePattern("%{COMMONAPACHELOG}", PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		b.Fatal(err)
	}

	line := `127.0.0.1 - - [23/Apr/2014:22:58:32 +0200] "GET /index.php HTTP/1.1" 404 207`
	pool := g.newStringBufferPool()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf, runErr := pool.run(g, line, true)
			if runErr != nil {
				b.Fatal(runErr)
			}
			if len(buf.Values) == 0 {
				b.Fatal("empty result")
			}
			pool.put(buf)
		}
	})
}

func BenchmarkRunWithTypeInfoWithPoolHelperParallel(b *testing.B) {
	g, err := CompilePattern("%{INT:A:int} %{WORD:B:bool} %{BASE10NUM:C:float}", PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		b.Fatal(err)
	}

	pool := g.newAnyBufferPool()
	line := "1 true 1.1"
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf, runErr := pool.run(g, line, true)
			if runErr != nil {
				b.Fatal(runErr)
			}
			if len(buf.Values) != 3 {
				b.Fatal(buf.Values)
			}
			pool.put(buf)
		}
	})
}

func BenchmarkRunStructuredComposite(b *testing.B) {
	pattern := `%{NOTSPACE:remote_addr} - %{NOTSPACE:remote_user} \[%{DATA:time_local}\] "%{WORD:method} %{NOTSPACE:request} HTTP/%{NUMBER:http_version}" %{INT:status} %{INT:body_bytes_sent}`
	g, err := CompilePattern(pattern, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		b.Fatal(err)
	}

	line := `127.0.0.1 - admin [23/Apr/2014:22:58:32 +0200] "GET /index.php HTTP/1.1" 404 207`
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		ret, runErr := g.Run(line, true)
		if runErr != nil {
			b.Fatal(runErr)
		}
		if len(ret) == 0 {
			b.Fatal("empty result")
		}
	}
}

func BenchmarkRunStructuredCompositeRegexpPath(b *testing.B) {
	pattern := `%{NOTSPACE:remote_addr} - %{NOTSPACE:remote_user} \[%{DATA:time_local}\] "%{WORD:method} %{NOTSPACE:request} HTTP/%{NUMBER:http_version}" %{INT:status} %{INT:body_bytes_sent}`
	g, err := CompilePattern(pattern, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		b.Fatal(err)
	}
	g.fastMatcher = nil

	line := `127.0.0.1 - admin [23/Apr/2014:22:58:32 +0200] "GET /index.php HTTP/1.1" 404 207`
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		ret, runErr := g.Run(line, true)
		if runErr != nil {
			b.Fatal(runErr)
		}
		if len(ret) == 0 {
			b.Fatal("empty result")
		}
	}
}

func BenchmarkRunStructuredSQLServer(b *testing.B) {
	pattern := `%{TIMESTAMP_ISO8601:time} %{NOTSPACE:origin}\s+%{GREEDYDATA:msg}`
	g, err := CompilePattern(pattern, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		b.Fatal(err)
	}

	line := `2024-06-21 09:14:18.123+00:00 sqlservr        Error: login failed for user test`
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		ret, runErr := g.Run(line, true)
		if runErr != nil {
			b.Fatal(runErr)
		}
		if len(ret) == 0 {
			b.Fatal("empty result")
		}
	}
}

func BenchmarkRunStructuredSQLServerRegexpPath(b *testing.B) {
	pattern := `%{TIMESTAMP_ISO8601:time} %{NOTSPACE:origin}\s+%{GREEDYDATA:msg}`
	g, err := CompilePattern(pattern, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		b.Fatal(err)
	}
	g.fastMatcher = nil

	line := `2024-06-21 09:14:18.123+00:00 sqlservr        Error: login failed for user test`
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		ret, runErr := g.Run(line, true)
		if runErr != nil {
			b.Fatal(runErr)
		}
		if len(ret) == 0 {
			b.Fatal("empty result")
		}
	}
}

func BenchmarkRunStructuredNginxAccess(b *testing.B) {
	pattern := `%{NOTSPACE:client_ip} %{NOTSPACE:http_ident} %{NOTSPACE:http_auth} \[%{HTTPDATE:time}\] "%{DATA:http_method} %{GREEDYDATA:http_url} HTTP/%{NUMBER:http_version}" %{INT:status_code} %{INT:bytes}`
	g, err := CompilePattern(pattern, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		b.Fatal(err)
	}

	line := `127.0.0.1 - admin [23/Apr/2014:22:58:32 +0200] "GET /index.php?a=1 HTTP/1.1" 404 207`
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		ret, runErr := g.Run(line, true)
		if runErr != nil {
			b.Fatal(runErr)
		}
		if len(ret) == 0 {
			b.Fatal("empty result")
		}
	}
}

func BenchmarkRunStructuredNginxAccessRegexpPath(b *testing.B) {
	pattern := `%{NOTSPACE:client_ip} %{NOTSPACE:http_ident} %{NOTSPACE:http_auth} \[%{HTTPDATE:time}\] "%{DATA:http_method} %{GREEDYDATA:http_url} HTTP/%{NUMBER:http_version}" %{INT:status_code} %{INT:bytes}`
	g, err := CompilePattern(pattern, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		b.Fatal(err)
	}
	g.fastMatcher = nil

	line := `127.0.0.1 - admin [23/Apr/2014:22:58:32 +0200] "GET /index.php?a=1 HTTP/1.1" 404 207`
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		ret, runErr := g.Run(line, true)
		if runErr != nil {
			b.Fatal(runErr)
		}
		if len(ret) == 0 {
			b.Fatal("empty result")
		}
	}
}

func BenchmarkRunStructuredLogLevel(b *testing.B) {
	pattern := `%{TIMESTAMP_ISO8601:time} \[%{LOGLEVEL:status}\] %{GREEDYDATA:msg}`
	g, err := CompilePattern(pattern, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		b.Fatal(err)
	}

	line := `2024-06-21 09:14:18+00:00 [error] database connection failed`
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		ret, runErr := g.Run(line, true)
		if runErr != nil {
			b.Fatal(runErr)
		}
		if len(ret) == 0 {
			b.Fatal("empty result")
		}
	}
}

func BenchmarkRunStructuredLogLevelRegexpPath(b *testing.B) {
	pattern := `%{TIMESTAMP_ISO8601:time} \[%{LOGLEVEL:status}\] %{GREEDYDATA:msg}`
	g, err := CompilePattern(pattern, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		b.Fatal(err)
	}
	g.fastMatcher = nil

	line := `2024-06-21 09:14:18+00:00 [error] database connection failed`
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		ret, runErr := g.Run(line, true)
		if runErr != nil {
			b.Fatal(runErr)
		}
		if len(ret) == 0 {
			b.Fatal("empty result")
		}
	}
}

func BenchmarkRunStructuredLogLevelUTF(b *testing.B) {
	pattern := `%{TIMESTAMP_ISO8601:time} \[%{LOGLEVEL:status}\] %{GREEDYDATA:msg}`
	g, err := CompilePattern(pattern, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		b.Fatal(err)
	}

	line := "2024-06-21 09:14:18+00:00 [INFO] 中文告警：磁盘空间不足，当前剩余空间 3%"
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		ret, runErr := g.Run(line, true)
		if runErr != nil {
			b.Fatal(runErr)
		}
		if len(ret) == 0 {
			b.Fatal("empty result")
		}
	}
}

func BenchmarkRunStructuredLogLevelUTFRegexpPath(b *testing.B) {
	pattern := `%{TIMESTAMP_ISO8601:time} \[%{LOGLEVEL:status}\] %{GREEDYDATA:msg}`
	g, err := CompilePattern(pattern, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		b.Fatal(err)
	}
	g.fastMatcher = nil

	line := "2024-06-21 09:14:18+00:00 [INFO] 中文告警：磁盘空间不足，当前剩余空间 3%"
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		ret, runErr := g.Run(line, true)
		if runErr != nil {
			b.Fatal(runErr)
		}
		if len(ret) == 0 {
			b.Fatal("empty result")
		}
	}
}

func BenchmarkRunStructuredSyslogLine(b *testing.B) {
	pattern := `%{SYSLOGTIMESTAMP:timestamp} %{SYSLOGHOST:logsource} %{GREEDYDATA:message}`
	g, err := CompilePattern(pattern, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		b.Fatal(err)
	}

	line := `Sep 18 19:30:23 derrick-ThinkPad-X230 consul[11803]: agent.server.connect: initialized primary datacenter`
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		ret, runErr := g.Run(line, true)
		if runErr != nil {
			b.Fatal(runErr)
		}
		if len(ret) == 0 {
			b.Fatal("empty result")
		}
	}
}

func BenchmarkRunStructuredSyslogLineRegexpPath(b *testing.B) {
	pattern := `%{SYSLOGTIMESTAMP:timestamp} %{SYSLOGHOST:logsource} %{GREEDYDATA:message}`
	g, err := CompilePattern(pattern, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		b.Fatal(err)
	}
	g.fastMatcher = nil

	line := `Sep 18 19:30:23 derrick-ThinkPad-X230 consul[11803]: agent.server.connect: initialized primary datacenter`
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		ret, runErr := g.Run(line, true)
		if runErr != nil {
			b.Fatal(runErr)
		}
		if len(ret) == 0 {
			b.Fatal("empty result")
		}
	}
}

func BenchmarkRunStructuredShortMismatch(b *testing.B) {
	pattern := `%{TIMESTAMP_ISO8601:time} \[%{LOGLEVEL:status}\] %{GREEDYDATA:msg}`
	g, err := CompilePattern(pattern, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		b.Fatal(err)
	}

	line := "x"
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		_, runErr := g.Run(line, true)
		if runErr != ErrMismatch {
			b.Fatalf("expected mismatch, got %v", runErr)
		}
	}
}

func BenchmarkRunStructuredShortMismatchRegexpPath(b *testing.B) {
	pattern := `%{TIMESTAMP_ISO8601:time} \[%{LOGLEVEL:status}\] %{GREEDYDATA:msg}`
	g, err := CompilePattern(pattern, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		b.Fatal(err)
	}
	g.fastMatcher = nil

	line := "x"
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		_, runErr := g.Run(line, true)
		if runErr != ErrMismatch {
			b.Fatalf("expected mismatch, got %v", runErr)
		}
	}
}

func benchmarkBuildCaptureMapNoPrealloc(g *GrokRegexp, content string, trimSpace bool) (map[string]string, error) {
	match, err := g.matchIndexes(content)
	if err != nil {
		return nil, err
	}

	return benchmarkAssembleCaptureMapNoPrealloc(g, content, match, trimSpace), nil
}

func benchmarkAssembleCaptureMapNoPrealloc(g *GrokRegexp, content string, match []int, trimSpace bool) map[string]string {
	captures := make(map[string]string)
	for i, name := range g.subMatchNames.name {
		captures[name] = extractMatch(content, match, g.subMatchNames.subexpIndex[i], trimSpace)
	}
	return captures
}

func benchmarkBuildCaptureMapNumSubexp(g *GrokRegexp, content string, trimSpace bool) (map[string]string, error) {
	match, err := g.matchIndexes(content)
	if err != nil {
		return nil, err
	}

	return benchmarkAssembleCaptureMapNumSubexp(g, content, match, trimSpace), nil
}

func benchmarkAssembleCaptureMapNumSubexp(g *GrokRegexp, content string, match []int, trimSpace bool) map[string]string {
	captures := make(map[string]string, g.re.NumSubexp())
	for i, name := range g.subMatchNames.name {
		captures[name] = extractMatch(content, match, g.subMatchNames.subexpIndex[i], trimSpace)
	}
	return captures
}

func benchmarkBuildCaptureMapNamedFields(g *GrokRegexp, content string, trimSpace bool) (map[string]string, error) {
	match, err := g.matchIndexes(content)
	if err != nil {
		return nil, err
	}

	return benchmarkAssembleCaptureMapNamedFields(g, content, match, trimSpace), nil
}

func benchmarkAssembleCaptureMapNamedFields(g *GrokRegexp, content string, match []int, trimSpace bool) map[string]string {
	captures := make(map[string]string, len(g.nameIndex))
	for i, name := range g.subMatchNames.name {
		captures[name] = extractMatch(content, match, g.subMatchNames.subexpIndex[i], trimSpace)
	}
	return captures
}

func BenchmarkBuildCaptureMapCapacityStrategy(b *testing.B) {
	cases := []struct {
		name    string
		pattern string
		line    string
	}{
		{
			name:    "COMMONAPACHELOG",
			pattern: `%{COMMONAPACHELOG}`,
			line:    `127.0.0.1 - - [23/Apr/2014:22:58:32 +0200] "GET /index.php HTTP/1.1" 404 207`,
		},
		{
			name:    "ElasticSearchDefault",
			pattern: `^\[%{TIMESTAMP_ISO8601:time}\]\[%{LOGLEVEL:status}%{SPACE}\]\[%{NOTSPACE:name}%{SPACE}\]%{SPACE}(\[%{HOSTNAME:nodeId}\])?.*`,
			line:    `[2021-06-01T11:45:15,927][WARN ][o.e.c.r.a.DiskThresholdMonitor] [master] high disk watermark [90%] exceeded on [A2kEFgMLQ1-vhMdZMJV3Iw][master][/tmp/elasticsearch-cluster/nodes/0] free: 17.1gb[7.3%], shards will be relocated away from this node; currently relocating away shards totalling [0] bytes; the node is expected to continue to exceed the high disk watermark when these relocations are complete`,
		},
	}

	for _, tc := range cases {
		tc := tc
		g, err := CompilePattern(tc.pattern, PatternStorage{defalutDenormalizedPatterns})
		if err != nil {
			b.Fatalf("%s: %v", tc.name, err)
		}

		b.Run(tc.name+"/no_prealloc", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				ret, err := benchmarkBuildCaptureMapNoPrealloc(g, tc.line, true)
				if err != nil {
					b.Fatal(err)
				}
				if len(ret) == 0 {
					b.Fatal("empty result")
				}
			}
		})

		b.Run(tc.name+"/num_subexp", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				ret, err := benchmarkBuildCaptureMapNumSubexp(g, tc.line, true)
				if err != nil {
					b.Fatal(err)
				}
				if len(ret) == 0 {
					b.Fatal("empty result")
				}
			}
		})

		b.Run(tc.name+"/named_fields", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				ret, err := benchmarkBuildCaptureMapNamedFields(g, tc.line, true)
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

func BenchmarkAssembleCaptureMapCapacityStrategy(b *testing.B) {
	cases := []struct {
		name    string
		pattern string
		line    string
	}{
		{
			name:    "COMMONAPACHELOG",
			pattern: `%{COMMONAPACHELOG}`,
			line:    `127.0.0.1 - - [23/Apr/2014:22:58:32 +0200] "GET /index.php HTTP/1.1" 404 207`,
		},
		{
			name:    "ElasticSearchDefault",
			pattern: `^\[%{TIMESTAMP_ISO8601:time}\]\[%{LOGLEVEL:status}%{SPACE}\]\[%{NOTSPACE:name}%{SPACE}\]%{SPACE}(\[%{HOSTNAME:nodeId}\])?.*`,
			line:    `[2021-06-01T11:45:15,927][WARN ][o.e.c.r.a.DiskThresholdMonitor] [master] high disk watermark [90%] exceeded on [A2kEFgMLQ1-vhMdZMJV3Iw][master][/tmp/elasticsearch-cluster/nodes/0] free: 17.1gb[7.3%], shards will be relocated away from this node; currently relocating away shards totalling [0] bytes; the node is expected to continue to exceed the high disk watermark when these relocations are complete`,
		},
	}

	for _, tc := range cases {
		tc := tc
		g, err := CompilePattern(tc.pattern, PatternStorage{defalutDenormalizedPatterns})
		if err != nil {
			b.Fatalf("%s: %v", tc.name, err)
		}
		match, err := g.matchIndexes(tc.line)
		if err != nil {
			b.Fatalf("%s: %v", tc.name, err)
		}

		b.Run(tc.name+"/no_prealloc", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				ret := benchmarkAssembleCaptureMapNoPrealloc(g, tc.line, match, true)
				if len(ret) == 0 {
					b.Fatal("empty result")
				}
			}
		})

		b.Run(tc.name+"/num_subexp", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				ret := benchmarkAssembleCaptureMapNumSubexp(g, tc.line, match, true)
				if len(ret) == 0 {
					b.Fatal("empty result")
				}
			}
		})

		b.Run(tc.name+"/named_fields", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				ret := benchmarkAssembleCaptureMapNamedFields(g, tc.line, match, true)
				if len(ret) == 0 {
					b.Fatal("empty result")
				}
			}
		})
	}
}
