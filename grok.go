// Package grok used to parses grok patterns in Go
package grok

import (
	"fmt"
	"regexp"
	"strings"

	regexpRust "github.com/BurntSushi/rure-go"

	regexpRe2 "github.com/gensliu/cre2-go"

	"github.com/spf13/cast"
)

var (
	valid    = regexp.MustCompile(`^\w+([-.]\w+)*(:([-.\w]+)(:(string|str|float|int|bool))?)?$`)
	normal   = regexp.MustCompile(`%{([\w-.]+(?::[\w-.]+(?::[\w-.]+)?)?)}`)
	symbolic = regexp.MustCompile(`\W`)
)

// Denormalized patterns as regular expressions.

type GrokRegexp struct {
	grokPattern *GrokPattern
	re          *regexpRust.Regex
	reStd       *regexp.Regexp
	re2         *regexpRe2.Regexp

	names map[string]int
}

func (g *GrokRegexp) RunStd(content interface{}, trimSpace bool) (map[string]string, error) {
	if g.reStd == nil {
		return nil, fmt.Errorf("not complied")
	}
	result := map[string]string{}

	switch v := content.(type) {
	case []byte:
		match := g.reStd.FindStringSubmatch(string(v))
		if len(match) == 0 {
			return nil, fmt.Errorf("no match")
		}
		for name, index := range g.names {
			if trimSpace {
				result[name] = strings.TrimSpace(match[index])
			} else {
				result[name] = match[index]
			}
		}
	case string:
		match := g.reStd.FindStringSubmatch(v)
		if len(match) == 0 {
			return nil, fmt.Errorf("no match")
		}
		for name, index := range g.names {
			if name != "" {
				if trimSpace {
					result[name] = strings.TrimSpace(match[index])
				} else {
					result[name] = match[index]
				}
			}
		}
	}
	return result, nil
}

func (g *GrokRegexp) RunRe2(content interface{}, trimSpace bool) (map[string]string, error) {
	if g.re2 == nil {
		return nil, fmt.Errorf("not complied")
	}
	result := map[string]string{}

	switch v := content.(type) {
	case []byte:
		matchA := g.re2.FindAllStringSubmatch(string(v), 1)
		if len(matchA) == 0 {
			return nil, fmt.Errorf("no match")
		}
		match := matchA[0]
		for name, index := range g.names {
			if trimSpace {
				result[name] = strings.TrimSpace(match[index])
			} else {
				result[name] = match[index]
			}
		}
	case string:
		matchA := g.re2.FindAllStringSubmatch(v, 1)
		if len(matchA) == 0 {
			return nil, fmt.Errorf("no match")
		}
		match := matchA[0]
		for name, index := range g.names {
			if name != "" {
				if trimSpace {
					result[name] = strings.TrimSpace(match[index])
				} else {
					result[name] = match[index]
				}
			}
		}
	}
	return result, nil
}

func (g *GrokRegexp) RunRust(content interface{}, trimSpace bool) (map[string]string, error) {
	if g.re == nil {
		return nil, fmt.Errorf("not complied")
	}
	result := map[string]string{}

	switch v := content.(type) {
	case []byte:
		c := g.re.NewCaptures()
		if !g.re.CapturesBytes(c, v) {
			return nil, fmt.Errorf("no match")
		}

		for name, index := range g.names {
			if name != "" {
				if s, e, ok := c.Group(index); ok {
					if trimSpace {
						result[name] = strings.TrimSpace((string(v[s:e])))
					} else {
						result[name] = string(v[s:e])
					}
				} else {
					result[name] = ""
				}
			}
		}
	case string:
		c := g.re.NewCaptures()
		if !g.re.Captures(c, v) {
			return nil, fmt.Errorf("no match")
		}

		for name, index := range g.names {
			if name != "" {
				if s, e, ok := c.Group(index); ok {
					if trimSpace {
						result[name] = strings.TrimSpace((v[s:e]))
					} else {
						result[name] = v[s:e]
					}
				} else {
					result[name] = ""
				}
			}
		}
	}
	return result, nil
}

func (g *GrokRegexp) RunWithTypeInfo(content interface{}, trimSpace bool) (map[string]interface{}, map[string]string, error) {
	castDst := map[string]interface{}{}
	castFail := map[string]string{}
	ret, err := g.RunRust(content, trimSpace)
	if err != nil {
		return nil, nil, err
	}
	var dstV interface{}
	for k, v := range ret {
		var err error
		dstV = v
		if varType, ok := g.grokPattern.varbType[k]; ok {
			switch varType {
			case GTypeInt:
				dstV, err = cast.ToInt64E(v)
			case GTypeFloat:
				dstV, err = cast.ToFloat64E(v)
			case GTypeBool:
				dstV, err = cast.ToBoolE(v)
			case GTypeStr:
			default:
				err = fmt.Errorf("unsupported data type: %s", varType)
			}
		}
		// TODO: use the default value of the data type
		// cast 操作失败赋予默认值
		castDst[k] = dstV
		if err != nil {
			castFail[k] = v
		}
	}
	return castDst, castFail, nil
}

func CompileGrokRegexp(input string, denomalized PatternsIface) (*GrokRegexp, error) {
	gP, err := DenormalizePattern(input, denomalized)
	if err != nil {
		return nil, err
	}
	re, err := regexpRust.Compile(gP.denormalized)
	if err != nil {
		return nil, err
	}

	names := map[string]int{}
	for index, name := range re.CaptureNames() {
		if name != "" {
			names[name] = index
		}
	}

	reStd, err := regexp.Compile(gP.denormalized)
	if err != nil {
		return nil, err
	}

	reRe2, err := regexpRe2.Compile(gP.denormalized)
	if err != nil {
		return nil, err
	}
	return &GrokRegexp{
		grokPattern: gP,
		re:          re,
		re2:         reRe2,
		reStd:       reStd,
		names:       names,
	}, nil
}
