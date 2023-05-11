package grok

import (
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
	g, err := CompileGrokRegexp("%{DAY:day}", Patterns{denormalized})
	if err != nil {
		t.Error(err)
	}
	ret, err := g.RunCgo("Tue qds", true)
	if err != nil {
		t.Error(err)
	}
	if ret["day"] != "Tue" {
		t.Fatalf("day should be 'Tue' have '%s'", ret["day"])
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
	g, err := CompileGrokRegexp("%{DAY:day}", Patterns{de})
	if err != nil {
		t.Error(err)
	}
	ret, err := g.RunCgo("Tue qds", true)
	if err != nil {
		t.Error(err)
	}
	if ret["day"] != "Tue" {
		t.Fatalf("day should be 'Tue' have '%s'", ret["day"])
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
		data      string
		ptn       string
		ret       map[string]interface{}
		failedRet map[string]string
		failed    bool
	}{
		{
			data: `1
true 1.1`,
			ptn: `%{INT:A:int}
%{WORD:B:bool} %{BASE10NUM:C:float}`,
			ret: map[string]interface{}{
				"A": int64(1),
				"B": true,
				"C": float64(1.1),
			},
			failedRet: map[string]string{},
		},
		{
			data: `1
true 1.1`,
			ptn: `%{INT:A:int}
%{WORD:B:bool} %{BASE10NUM:C:int}`,
			ret: map[string]interface{}{
				"A": int64(1),
				"B": true,
				"C": int64(0),
			},
			failedRet: map[string]string{
				"C": "1.1",
			},
		},
		{
			data: `1 ijk123abc
true 1.1`,
			ptn: `%{INT:A:int} %{WORD:S:string}
%{WORD:B:bool} %{BASE10NUM:C:int}`,
			ret: map[string]interface{}{
				"A": int64(1),
				"S": "ijk123abc",
				"B": true,
				"C": int64(0),
			},
			failedRet: map[string]string{
				"C": "1.1",
			},
		},
		{
			data: `1
true 1.1`,
			ptn: `%{INT:A}
%{WORD:B:bool} %{BASE10NUM:C:int}`,
			ret: map[string]interface{}{
				"A": "1",
				"B": true,
				"C": int64(0),
			},
			failedRet: map[string]string{
				"C": "1.1",
			},
		},
	}

	for _, item := range tCase {
		g, err := CompileGrokRegexp(item.ptn, Patterns{defalutDenormalizedPatterns})
		if err != nil {
			t.Fatal(err)
		}
		v, vf, err := g.RunWithTypeInfo(item.data, true)
		if err != nil && !item.failed {
			t.Fatal(err)
		}
		assert.Equal(t, item.ret, v)
		assert.Equal(t, item.failedRet, vf)
	}
}

func BenchmarkReCgo(b *testing.B) {
	pathPatterns, err := LoadPatternsFromPath("./patterns")
	if err != nil {
		b.Fatal(err)
	}
	de, errs := DenormalizePatternsFromMap(pathPatterns)
	if len(errs) != 0 {
		b.Fatal(errs)
	}

	p := "\\d{4}(?:-\\d{2}){2} \\d{2}(?:\\:\\d{2}){2}\\.\\d+\\s+\\[%{NOTSPACE:param2}\\]\\s+%{WORD:param3}\\s+[\\.a-zA-Z0-9]+\\s+(?:-|(?P<param5>.+?))\\s+\\[%{NOTSPACE:param6}\\]\\s+%{NOTSPACE:param7}\\s+" +
		"(?P<param8>\\d+)\\s+(?P<param9>\\d+)\\s+(?:-|(?P<param10>.+?))\\s+"
	g, err := CompileGrokRegexp(p, Patterns{de})
	if err != nil {
		b.Fatal(err)
	}

	data := `2023-04-11 18:00:11.428 [ForkJoinPool.commonPool-worker-6] INFO  com.crt.loan.imc.es.impl.EsServiceImpl - [lambda$insert$0,191] loan-cmis-imc 2252660116907431929 672358609369876840   - [影像迁移]-mq信息推送成功,[{"applSeq":"1026202304111000489","applType":"C","channelId":"26","createTime":1681207211000,"custId":"889001302821","downloadAddress":"/opt/loan-fileload-service/imc/../ycms/20260701/1026202304111000489/fec3f5ef363e47daa9264a95d38866d3_007.jpg","esId":"7HjCb4cB-yoLUoESNeZv","filePath":"group2/M00/05/54/ChIB8mQ1L6uAPWyrAAdmZPIYOl0583.jpg","fileSourceType":"N","id":40,"idNo":"120223198208301638","idType":"20","imgType":"ID_PHOTO","outerNo":"1681207211275","productId":"PPD005","requestNo":"1645728441670459392","signType":"N","status":"SUCCESS","updateTime":1681207211425}]`

	for n := 0; n < b.N; n++ {
		if _, err := g.RunCgo(data, true); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReCgo2(b *testing.B) {
	pathPatterns, err := LoadPatternsFromPath("./patterns")
	if err != nil {
		b.Fatal(err)
	}
	de, errs := DenormalizePatternsFromMap(pathPatterns)
	if len(errs) != 0 {
		b.Fatal(errs)
	}

	p := "%{IPORHOST:client_ip} %{NOTSPACE:http_ident} %{NOTSPACE:http_auth} \\[%{HTTPDATE:time}\\] \"%{DATA:http_method} %{GREEDYDATA:http_url} HTTP/%{NUMBER:http_version}\" %{INT:status_code} %{INT:bytes}"
	g, err := CompileGrokRegexp(p, Patterns{de})
	if err != nil {
		b.Fatal(err)
	}

	data := `127.0.0.1 - - [21/Jul/2021:14:14:38 +0800] "GET /?1 HTTP/1.1" 200 2178 "-" "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.164 Safari/537.36"`
	for n := 0; n < b.N; n++ {
		if _, err := g.RunCgo(data, true); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReCgo3(b *testing.B) {
	pathPatterns, err := LoadPatternsFromPath("./patterns")
	if err != nil {
		b.Fatal(err)
	}
	de, errs := DenormalizePatternsFromMap(pathPatterns)
	if len(errs) != 0 {
		b.Fatal(errs)
	}

	p := "%{NOTSPACE:client_ip} %{NOTSPACE:http_ident} %{NOTSPACE:http_auth} \\[%{HTTPDATE:time}\\] \"%{DATA:http_method} %{GREEDYDATA:http_url} HTTP/%{NUMBER:http_version}\" %{INT:status_code} %{INT:bytes}"
	g, err := CompileGrokRegexp(p, Patterns{de})
	if err != nil {
		b.Fatal(err)
	}

	data := `127.0.0.1 - - [21/Jul/2021:14:14:38 +0800] "GET /?1 HTTP/1.1" 200 2178 "-" "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.164 Safari/537.36"`
	for n := 0; n < b.N; n++ {
		if _, err := g.RunCgo(data, true); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReStd(b *testing.B) {
	pathPatterns, err := LoadPatternsFromPath("./patterns")
	if err != nil {
		b.Fatal(err)
	}
	de, errs := DenormalizePatternsFromMap(pathPatterns)
	if len(errs) != 0 {
		b.Fatal(errs)
	}

	p := "\\d{4}(?:-\\d{2}){2} \\d{2}(?:\\:\\d{2}){2}\\.\\d+\\s+\\[%{NOTSPACE:param2}\\]\\s+%{WORD:param3}\\s+[\\.a-zA-Z0-9]+\\s+(?:-|(?P<param5>.+?))\\s+\\[%{NOTSPACE:param6}\\]\\s+%{NOTSPACE:param7}\\s+" +
		"(?P<param8>\\d+)\\s+(?P<param9>\\d+)\\s+(?:-|(?P<param10>.+?))\\s+"
	g, err := CompileGrokRegexp(p, Patterns{de})
	if err != nil {
		b.Fatal(err)
	}

	data := `2023-04-11 18:00:11.428 [ForkJoinPool.commonPool-worker-6] INFO  com.crt.loan.imc.es.impl.EsServiceImpl - [lambda$insert$0,191] loan-cmis-imc 2252660116907431929 672358609369876840   - [影像迁移]-mq信息推送成功,[{"applSeq":"1026202304111000489","applType":"C","channelId":"26","createTime":1681207211000,"custId":"889001302821","downloadAddress":"/opt/loan-fileload-service/imc/../ycms/20260701/1026202304111000489/fec3f5ef363e47daa9264a95d38866d3_007.jpg","esId":"7HjCb4cB-yoLUoESNeZv","filePath":"group2/M00/05/54/ChIB8mQ1L6uAPWyrAAdmZPIYOl0583.jpg","fileSourceType":"N","id":40,"idNo":"120223198208301638","idType":"20","imgType":"ID_PHOTO","outerNo":"1681207211275","productId":"PPD005","requestNo":"1645728441670459392","signType":"N","status":"SUCCESS","updateTime":1681207211425}]`

	for n := 0; n < b.N; n++ {
		if _, err := g.RunStd(data, true); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReStd2(b *testing.B) {
	pathPatterns, err := LoadPatternsFromPath("./patterns")
	if err != nil {
		b.Fatal(err)
	}
	de, errs := DenormalizePatternsFromMap(pathPatterns)
	if len(errs) != 0 {
		b.Fatal(errs)
	}

	p := "%{IPORHOST:client_ip} %{NOTSPACE:http_ident} %{NOTSPACE:http_auth} \\[%{HTTPDATE:time}\\] \"%{DATA:http_method} %{GREEDYDATA:http_url} HTTP/%{NUMBER:http_version}\" %{INT:status_code} %{INT:bytes}"
	g, err := CompileGrokRegexp(p, Patterns{de})
	if err != nil {
		b.Fatal(err)
	}

	data := `127.0.0.1 - - [21/Jul/2021:14:14:38 +0800] "GET /?1 HTTP/1.1" 200 2178 "-" "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.164 Safari/537.36"`
	for n := 0; n < b.N; n++ {
		if _, err := g.RunStd(data, true); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReStd3(b *testing.B) {
	pathPatterns, err := LoadPatternsFromPath("./patterns")
	if err != nil {
		b.Fatal(err)
	}
	de, errs := DenormalizePatternsFromMap(pathPatterns)
	if len(errs) != 0 {
		b.Fatal(errs)
	}

	p := "%{NOTSPACE:client_ip} %{NOTSPACE:http_ident} %{NOTSPACE:http_auth} \\[%{HTTPDATE:time}\\] \"%{DATA:http_method} %{GREEDYDATA:http_url} HTTP/%{NUMBER:http_version}\" %{INT:status_code} %{INT:bytes}"
	g, err := CompileGrokRegexp(p, Patterns{de})
	if err != nil {
		b.Fatal(err)
	}

	data := `127.0.0.1 - - [21/Jul/2021:14:14:38 +0800] "GET /?1 HTTP/1.1" 200 2178 "-" "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.164 Safari/537.36"`
	for n := 0; n < b.N; n++ {
		if _, err := g.RunStd(data, true); err != nil {
			b.Fatal(err)
		}
	}
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
