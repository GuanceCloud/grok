package grok

import "testing"

func FuzzStructuredApacheErrorParity(f *testing.F) {
	current, err := CompilePattern(`\[%{HTTPDERROR_DATE:time}\] \[%{GREEDYDATA:type}:%{GREEDYDATA:status}\] \[pid %{GREEDYDATA:pid}:tid %{GREEDYDATA:tid}\] `, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		f.Fatal(err)
	}
	regexpOnly, err := CompilePattern(`\[%{HTTPDERROR_DATE:time}\] \[%{GREEDYDATA:type}:%{GREEDYDATA:status}\] \[pid %{GREEDYDATA:pid}:tid %{GREEDYDATA:tid}\] `, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		f.Fatal(err)
	}
	regexpOnly.fastMatcher = nil
	regexpOnly.prefilter = nil

	for _, seed := range []string{
		`[Wed Jun 02 16:32:14.123456 2021] [authz_core:error] [pid 1234:tid 140735] `,
		`[Wed Jun 02 16:32:14.123456 2021] [authz_core:warn] [pid 1234:tid 140735] `,
		`[Wed Jun 02 16:32:14.123456 2021] [authz_core:error] [pid 1234] `,
		``,
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, line string) {
		assertRunParity(t, current, regexpOnly, line, true)
	})
}

func FuzzStructuredConsulFixtureParity(f *testing.F) {
	fixture := mustDatakitFixtureByName(f, "consul/consul/consul/Consul_log")
	regexpOnly := rawRegexpCopy(fixture.regexpOnly)
	for _, seed := range []string{
		fixture.line,
		fixture.line + "\n",
		`Sep 18 19:30:23 derrick-ThinkPad-X230 consul[11803]: 2021-09-18T19:30:23.522+0800 [WARN]  agent.server.connect: initialized primary datacenter`,
		`Sep 18 19:30:23 derrick-ThinkPad-X230 consul[11803]:`,
		``,
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, line string) {
		assertRunParity(t, fixture.current, regexpOnly, line, true)
	})
}

func FuzzStructuredMySQLSlowFixtureParity(f *testing.F) {
	fixture := mustDatakitFixtureByName(f, "mysql/mysql/mysql/MySQL_slow_log")
	regexpOnly := rawRegexpCopy(fixture.regexpOnly)
	for _, seed := range []string{
		fixture.line,
		fixture.line + "\n",
		`# Time: 2019-11-27T10:43:13.460744Z
# User@Host: root[root] @ localhost [1.2.3.4]  Id:    35
# Query_time: 0.214922  Lock_time: 0.000094 Rows_sent: 1  Rows_examined: 123456
SET timestamp=1574851393;
SELECT 1`,
		`# Time: `,
		``,
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, line string) {
		assertRunParity(t, fixture.current, regexpOnly, line, true)
	})
}

func FuzzStructuredElasticSearchDefaultFixtureParity(f *testing.F) {
	fixture := mustDatakitFixtureByName(f, "elasticsearch/elasticsearch/elasticsearch/ElasticSearch_log")
	regexpOnly := rawRegexpCopy(fixture.regexpOnly)
	for _, seed := range []string{
		fixture.line,
		`[2021-06-01T11:45:15,927][WARN ][o.e.c.r.a.DiskThresholdMonitor] [master] high disk watermark [90%] exceeded`,
		`[2021-06-01T11:45:15,927][WARN ][o.e.c.r.a.DiskThresholdMonitor] high disk watermark [90%] exceeded`,
		`[2021-06-01T11:45:15,927][WARN ][o.e.c.r.a.DiskThresholdMonitor] [] high disk watermark [90%] exceeded`,
		``,
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, line string) {
		assertRunParity(t, fixture.current, regexpOnly, line, true)
	})
}

func FuzzStructuredNginxAccessFixtureParity(f *testing.F) {
	fixture := mustDatakitFixtureByName(f, "nginx/nginx/nginx/Nginx_access_log")
	regexpOnly := rawRegexpCopy(fixture.regexpOnly)
	for _, seed := range []string{
		fixture.line,
		`127.0.0.1 - - [24/Mar/2021:13:54:19 +0800] "GET /basic_status HTTP/1.1" 200 97 "-" "Mozilla/5.0"`,
		`127.0.0.1 x - [24/Mar/2021:13:54:19 +0800] "GET /basic_status HTTP/1.1" 200 97 "-" "Mozilla/5.0"`,
		`127.0.0.1 - x [24/Mar/2021:13:54:19 +0800] "GET /basic_status HTTP/1.1" 200 97 "-" "Mozilla/5.0"`,
		``,
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, line string) {
		assertRunParity(t, fixture.current, regexpOnly, line, true)
	})
}

func FuzzStructuredTomcatCatalinaFixtureParity(f *testing.F) {
	fixture := mustDatakitFixtureByName(f, "tomcat/tomcat/tomcat/Tomcat_Catalina_log")
	regexpOnly := rawRegexpCopy(fixture.regexpOnly)
	for _, seed := range []string{
		fixture.line,
		`06-Sep-2021 22:33:30.513 INFO [main] org.apache.catalina.startup.VersionLoggerListener.log Command line argument: -Xmx256m`,
		`06-Sep-2021 22:33:30.513 INFO [] org.apache.catalina.startup.VersionLoggerListener.log Command line argument: -Xmx256m`,
		`06-Sep-2021 22:33:30.513 INFO [main] x`,
		``,
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, line string) {
		assertRunParity(t, fixture.current, regexpOnly, line, true)
	})
}

func FuzzRegexpPrefilterLiteralSetParity(f *testing.F) {
	current, err := CompilePattern(`^(?:alpha|bravo|charlie|delta|echo|foxtrot|golf|hotel|india|juliet|kilo|lima|mike|november|oscar|papa)$`, PatternStorage{defalutDenormalizedPatterns})
	if err != nil {
		f.Fatal(err)
	}
	current.fastMatcher = nil

	regexpOnly := *current
	regexpOnly.prefilter = nil

	for _, seed := range []string{
		"alpha",
		"bravo",
		"xxbravoyy",
		"zzzz",
		"",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, line string) {
		assertRunParity(t, current, &regexpOnly, line, true)
	})
}

func mustDatakitFixtureByName(t testing.TB, name string) datakitMatchedFixture {
	t.Helper()
	fixtures, _ := loadMatchedDatakitFixtures(t)
	for _, fixture := range fixtures {
		if fixture.name == name {
			return fixture
		}
	}
	t.Fatalf("expected datakit fixture %q", name)
	return datakitMatchedFixture{}
}

func rawRegexpCopy(g *GrokRegexp) *GrokRegexp {
	copy := *g
	copy.fastMatcher = nil
	copy.prefilter = nil
	return &copy
}
