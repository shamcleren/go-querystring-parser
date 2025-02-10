%{
package querystring
%}

%union {
	s 	string
	n 	int
	q 	Condition
}

%token tSTRING tPHRASE tNUMBER tSLASH tSTAR
%token tOR tAND tNOT tTO tPLUS tMINUS tCOLON
%token tLEFTBRACKET tRIGHTBRACKET tLEFTRANGE tRIGHTRANGE tLEFTBRACES tRIGHTBRACES
%token tGREATER tLESS tEQUAL

%type <s>                tSTRING
%type <s>                tPHRASE
%type <s>                tNUMBER
%type <s>                tSTAR
%type <s>		 tSLASH
%type <s>                posOrNegNumber
%type <q>                searchBase searchLogicParts searchPart searchLogicPart searchLogicSimplePart
%type <n>                searchPrefix

%left tOR
%left tAND
%nonassoc tLEFTBRACKET tRIGHTBRACKET

%%

input:
searchLogicParts {
	yylex.(*lexerWrapper).query = $1
};

searchLogicParts:
searchLogicPart searchLogicParts {
	$$ = NewAndCondition($1, $2)
}
|
searchLogicPart {
	$$ = $1
}

searchLogicPart:
searchLogicSimplePart {
	$$ = $1
}
|
searchLogicSimplePart tOR searchLogicPart {
	$$ = NewOrCondition($1, $3)
}
|
searchLogicSimplePart tAND searchLogicPart {
	$$ = NewAndCondition($1, $3)
};

searchLogicSimplePart:
searchPart {
	$$ = $1
}
|
tLEFTBRACKET searchLogicPart tRIGHTBRACKET {
	$$ = $2
}
|
tNOT searchLogicSimplePart {
	$$ = NewNotCondition($2)
};

searchPart:
searchPrefix searchBase {
	switch($1) {
	case queryMustNot:
		$$ = NewNotCondition($2)
	default:
		$$ = $2
	}
}
|
searchBase {
	$$ = $1
};

searchPrefix:
tPLUS {
	$$ = queryMust
}
|
tMINUS {
	$$ = queryMustNot
};

searchBase:
tSTRING {
	$$ = newStringCondition($1)
}
|
tNUMBER {
	$$ = NewMatchCondition($1)
}
|
tPHRASE {
	phrase := $1
	q := NewMatchCondition(phrase)
	$$ = q
}
|
tSLASH{
	phrase := $1
	q := NewRegexpCondition(phrase)
	$$ = q
}
|
tSTRING tCOLON tSTRING {
	q := newStringCondition($3)
	q.SetField($1)
	$$ = q
}
|
tSTRING tCOLON tLEFTBRACKET tSTRING tRIGHTBRACKET {
	q := newStringCondition($4)
	q.SetField($1)
	$$ = q
}
|
tSTRING tCOLON posOrNegNumber {
	q := NewMatchCondition($3)
	q.SetField($1)
	$$ = q
}
|
tSTRING tCOLON tPHRASE {
	q := NewMatchCondition($3)
	q.SetField($1)
	$$ = q
}
|
tSTRING tCOLON tSLASH {
	q := NewRegexpCondition($3)
	q.SetField($1)
	$$ = q
}
|
tSTRING tCOLON tGREATER posOrNegNumber {
	val := $4
	q := NewNumberRangeCondition(&val, nil, false, false)
	q.SetField($1)
	$$ = q
}
|
tSTRING tCOLON tGREATER tEQUAL posOrNegNumber {
	val := $5
	q := NewNumberRangeCondition(&val, nil, true, false)
	q.SetField($1)
	$$ = q
}
|
tSTRING tCOLON tLESS posOrNegNumber {
	val := $4
	q := NewNumberRangeCondition(nil, &val, false, false)
	q.SetField($1)
	$$ = q
}
|
tSTRING tCOLON tLESS tEQUAL posOrNegNumber {
	val := $5
	q := NewNumberRangeCondition(nil, &val, false, true)
	q.SetField($1)
	$$ = q
}
|
tSTRING tCOLON tGREATER tPHRASE {
	phrase := $4
	q := NewTimeRangeCondition(&phrase, nil, false, false)
	q.SetField($1)
	$$ = q
}
|
tSTRING tCOLON tGREATER tEQUAL tPHRASE {
	phrase := $5
	q := NewTimeRangeCondition(&phrase, nil, true, false)
	q.SetField($1)
	$$ = q
}
|
tSTRING tCOLON tLESS tPHRASE {
	phrase := $4
	q := NewTimeRangeCondition(nil, &phrase, false, false)
	q.SetField($1)
	$$ = q
}
|
tSTRING tCOLON tLESS tEQUAL tPHRASE {
	phrase := $5
	q := NewTimeRangeCondition(nil, &phrase, false, true)
	q.SetField($1)
	$$ = q
}
|
tSTRING tCOLON tLEFTRANGE tSTAR tTO posOrNegNumber tRIGHTRANGE {
	max := $6
	q := NewNumberRangeCondition(nil, &max, true, true)
	q.SetField($1)
	$$ = q
}
|
tSTRING tCOLON tLEFTRANGE posOrNegNumber tTO tSTAR tRIGHTRANGE {
	min := $4
	q := NewNumberRangeCondition(&min, nil, true, true)
	q.SetField($1)
	$$ = q
}
|
tSTRING tCOLON tLEFTRANGE posOrNegNumber tTO posOrNegNumber tRIGHTBRACES {
	min := $4
	max := $6
	q := NewNumberRangeCondition(&min, &max, true, false)
	q.SetField($1)
	$$ = q
}
|
tSTRING tCOLON tLEFTRANGE tPHRASE tTO tPHRASE tRIGHTBRACES {
	min := $4
	max := $6
	q := NewTimeRangeCondition(&min, &max, true, false)
	q.SetField($1)
	$$ = q
}
|
tSTRING tCOLON tLEFTBRACES posOrNegNumber tTO posOrNegNumber tRIGHTRANGE {
	min := $4
	max := $6
	q := NewNumberRangeCondition(&min, &max, false, true)
	q.SetField($1)
	$$ = q
}
|
tSTRING tCOLON tLEFTBRACES tPHRASE tTO tPHRASE tRIGHTRANGE {
	min := $4
	max := $6
	q := NewTimeRangeCondition(&min, &max, false, true)
	q.SetField($1)
	$$ = q
}
|
tSTRING tCOLON tLEFTRANGE posOrNegNumber tTO posOrNegNumber tRIGHTRANGE {
	min := $4
	max := $6
	q := NewNumberRangeCondition(&min, &max, true, true)
	q.SetField($1)
	$$ = q
}
|
tSTRING tCOLON tLEFTRANGE tPHRASE tTO tPHRASE tRIGHTRANGE {
	min := $4
	max := $6
	q := NewTimeRangeCondition(&min, &max, true, true)
	q.SetField($1)
	$$ = q
};

posOrNegNumber:
tNUMBER {
	$$ = $1
}
|
tMINUS tNUMBER {
	$$ = "-" + $2
};
