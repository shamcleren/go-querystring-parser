package querystring

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParser(t *testing.T) {
	testCases := map[string]struct {
		q string
		e Condition
	}{
		"正常查询": {
			q: `test`,
			e: &MatchCondition{
				Value: `test`,
			},
		},
		"负数查询": {
			q: `-test`,
			e: &NotCondition{
				Condition: &MatchCondition{
					Value: `test`,
				},
			},
		},
		"负数查询多条件": {
			q: `-test AND good`,
			e: &AndCondition{
				Left: &NotCondition{
					Condition: &MatchCondition{
						Value: `test`,
					},
				},
				Right: &MatchCondition{
					Value: `good`,
				},
			},
		},
		"通配符匹配": {
			q: `qu?ck bro*`,
			e: &AndCondition{
				Left: &WildcardCondition{
					Value: "qu?ck",
				},
				Right: &WildcardCondition{
					Value: "bro*",
				},
			},
		},
		"正则匹配": {
			q: `name:/joh?n(ath[oa]n)/`,
			e: &RegexpCondition{
				Field: "name",
				Value: "joh?n(ath[oa]n)",
			},
		},
		"范围匹配，左闭右开": {
			q: `count:[1 TO 5}`,
			e: &NumberRangeCondition{
				Field:        "count",
				Start:        pointer("1"),
				End:          pointer("5"),
				IncludeStart: true,
				IncludeEnd:   false,
			},
		},
		"范围匹配": {
			q: `count:[1 TO 5]`,
			e: &NumberRangeCondition{
				Field:        "count",
				Start:        pointer("1"),
				End:          pointer("5"),
				IncludeStart: true,
				IncludeEnd:   true,
			},
		},
		"范围匹配（无上限）": {
			q: `count:[10 TO *]`,
			e: &NumberRangeCondition{
				Field:        "count",
				Start:        pointer("10"),
				IncludeStart: true,
				IncludeEnd:   true,
			},
		},
		"字段匹配": {
			q: `status:active`,
			e: &MatchCondition{
				Field: "status",
				Value: "active",
			},
		},
		"字段匹配 + 括号": {
			q: `status:(active)`,
			e: &MatchCondition{
				Field: "status",
				Value: "active",
			},
		},
		"多条件组合，括号调整优先级": {
			q: `author:"John Smith" AND (age:20 OR status:active)`,
			e: &AndCondition{
				Left: &MatchCondition{
					Field: "author",
					Value: "John Smith",
				},
				Right: &OrCondition{
					Left: &MatchCondition{
						Field: "age",
						Value: "20",
					},
					Right: &MatchCondition{
						Field: "status",
						Value: "active",
					},
				},
			},
		},
		"多条件组合，and 和 or 的优先级": {
			q: `(author:"John Smith" AND age:20) OR status:active`,
			e: &OrCondition{
				Left: &AndCondition{
					Left: &MatchCondition{
						Field: "author",
						Value: "John Smith",
					},
					Right: &MatchCondition{
						Field: "age",
						Value: "20",
					},
				},
				Right: &MatchCondition{
					Field: "status",
					Value: "active",
				},
			},
		},
		"嵌套逻辑表达式": {
			q: `a:1 AND (b:2 OR c:3)`,
			e: &AndCondition{
				Left: &MatchCondition{
					Field: "a",
					Value: "1",
				},
				Right: &OrCondition{
					Left: &MatchCondition{
						Field: "b",
						Value: "2",
					},
					Right: &MatchCondition{
						Field: "c",
						Value: "3",
					},
				},
			},
		},
		"模糊匹配": {
			q: `quick brown fox`,
			e: &AndCondition{
				Left: &MatchCondition{
					Value: "quick",
				},
				Right: &AndCondition{
					Left: &MatchCondition{
						Value: "brown",
					},
					Right: &MatchCondition{
						Value: "fox",
					},
				},
			},
		},
		"单个条件精确匹配": {
			q: `log: "ERROR MSG"`,
			e: &MatchCondition{
				Field: "log",
				Value: "ERROR MSG",
			},
		},
		"match and time range": {
			q: "message: test\\ value AND datetime: [\"2020-01-01T00:00:00\" TO \"2020-12-31T00:00:00\"]",
			e: &AndCondition{
				Left: &MatchCondition{
					Field: "message",
					Value: "test value",
				},
				Right: &TimeRangeCondition{
					Field:        "datetime",
					Start:        pointer("2020-01-01T00:00:00"),
					End:          pointer("2020-12-31T00:00:00"),
					IncludeStart: true,
					IncludeEnd:   true,
				},
			},
		},
		"mixed or / and": {
			q: "a: 1 OR (b: 2 and c: 4)",
			e: &OrCondition{
				Left: &MatchCondition{
					Field: "a",
					Value: "1",
				},
				Right: &AndCondition{
					Left: &MatchCondition{
						Field: "b",
						Value: "2",
					},
					Right: &MatchCondition{
						Field: "c",
						Value: "4",
					},
				},
			},
		},
	}

	debugLexer = true

	for name, c := range testCases {
		t.Run(name, func(t *testing.T) {
			cond, err := Parse(c.q)
			if err != nil {
				t.Errorf("parse return error, %s", err)
				return
			}
			assert.Equal(t, c.e, cond)
		})
	}
}
func pointer(s string) *string {
	return &s
}
