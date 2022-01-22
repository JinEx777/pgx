package pgtype_test

import (
	"context"
	"encoding/json"
	"math"
	"math/big"
	"math/rand"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgtype/testutil"
	"github.com/stretchr/testify/require"
)

func mustParseBigInt(t *testing.T, src string) *big.Int {
	i := &big.Int{}
	if _, ok := i.SetString(src, 10); !ok {
		t.Fatalf("could not parse big.Int: %s", src)
	}
	return i
}

func isExpectedEqNumeric(a interface{}) func(interface{}) bool {
	return func(v interface{}) bool {
		aa := a.(pgtype.Numeric)
		vv := v.(pgtype.Numeric)

		if aa.Valid != vv.Valid {
			return false
		}

		// If NULL doesn't matter what the rest of the values are.
		if !aa.Valid {
			return true
		}

		if !(aa.NaN == vv.NaN && aa.InfinityModifier == vv.InfinityModifier) {
			return false
		}

		// If NaN or InfinityModifier are set then Int and Exp don't matter.
		if aa.NaN || aa.InfinityModifier != pgtype.None {
			return true
		}

		aaInt := (&big.Int{}).Set(aa.Int)
		vvInt := (&big.Int{}).Set(vv.Int)

		if aa.Exp < vv.Exp {
			mul := (&big.Int{}).Exp(big.NewInt(10), big.NewInt(int64(vv.Exp-aa.Exp)), nil)
			vvInt.Mul(vvInt, mul)
		} else if aa.Exp > vv.Exp {
			mul := (&big.Int{}).Exp(big.NewInt(10), big.NewInt(int64(aa.Exp-vv.Exp)), nil)
			aaInt.Mul(aaInt, mul)
		}

		return aaInt.Cmp(vvInt) == 0
	}
}

func mustParseNumeric(t *testing.T, src string) pgtype.Numeric {
	var n pgtype.Numeric
	plan := pgtype.NumericCodec{}.PlanScan(nil, pgtype.NumericOID, pgtype.TextFormatCode, &n, false)
	require.NotNil(t, plan)
	err := plan.Scan([]byte(src), &n)
	require.NoError(t, err)
	return n
}

func TestNumericCodec(t *testing.T) {
	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(147454), nil)
	max.Add(max, big.NewInt(1))
	longestNumeric := pgtype.Numeric{Int: max, Exp: -16383, Valid: true}

	testutil.RunTranscodeTests(t, "numeric", []testutil.TranscodeTestCase{
		{mustParseNumeric(t, "1"), new(pgtype.Numeric), isExpectedEqNumeric(mustParseNumeric(t, "1"))},
		{mustParseNumeric(t, "3.14159"), new(pgtype.Numeric), isExpectedEqNumeric(mustParseNumeric(t, "3.14159"))},
		{mustParseNumeric(t, "100010001"), new(pgtype.Numeric), isExpectedEqNumeric(mustParseNumeric(t, "100010001"))},
		{mustParseNumeric(t, "100010001.0001"), new(pgtype.Numeric), isExpectedEqNumeric(mustParseNumeric(t, "100010001.0001"))},
		{mustParseNumeric(t, "4237234789234789289347892374324872138321894178943189043890124832108934.43219085471578891547854892438945012347981"), new(pgtype.Numeric), isExpectedEqNumeric(mustParseNumeric(t, "4237234789234789289347892374324872138321894178943189043890124832108934.43219085471578891547854892438945012347981"))},
		{mustParseNumeric(t, "0.8925092023480223478923478978978937897879595901237890234789243679037419057877231734823098432903527585734549035904590854890345905434578345789347890402348952348905890489054234237489234987723894789234"), new(pgtype.Numeric), isExpectedEqNumeric(mustParseNumeric(t, "0.8925092023480223478923478978978937897879595901237890234789243679037419057877231734823098432903527585734549035904590854890345905434578345789347890402348952348905890489054234237489234987723894789234"))},
		{mustParseNumeric(t, "0.000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000123"), new(pgtype.Numeric), isExpectedEqNumeric(mustParseNumeric(t, "0.000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000123"))},
		{pgtype.Numeric{Int: mustParseBigInt(t, "243723409723490243842378942378901237502734019231380123"), Exp: 23790, Valid: true}, new(pgtype.Numeric), isExpectedEqNumeric(pgtype.Numeric{Int: mustParseBigInt(t, "243723409723490243842378942378901237502734019231380123"), Exp: 23790, Valid: true})},
		{pgtype.Numeric{Int: mustParseBigInt(t, "2437"), Exp: 23790, Valid: true}, new(pgtype.Numeric), isExpectedEqNumeric(pgtype.Numeric{Int: mustParseBigInt(t, "2437"), Exp: 23790, Valid: true})},
		{pgtype.Numeric{Int: mustParseBigInt(t, "43723409723490243842378942378901237502734019231380123"), Exp: 80, Valid: true}, new(pgtype.Numeric), isExpectedEqNumeric(pgtype.Numeric{Int: mustParseBigInt(t, "43723409723490243842378942378901237502734019231380123"), Exp: 80, Valid: true})},
		{pgtype.Numeric{Int: mustParseBigInt(t, "43723409723490243842378942378901237502734019231380123"), Exp: 81, Valid: true}, new(pgtype.Numeric), isExpectedEqNumeric(pgtype.Numeric{Int: mustParseBigInt(t, "43723409723490243842378942378901237502734019231380123"), Exp: 81, Valid: true})},
		{pgtype.Numeric{Int: mustParseBigInt(t, "43723409723490243842378942378901237502734019231380123"), Exp: 82, Valid: true}, new(pgtype.Numeric), isExpectedEqNumeric(pgtype.Numeric{Int: mustParseBigInt(t, "43723409723490243842378942378901237502734019231380123"), Exp: 82, Valid: true})},
		{pgtype.Numeric{Int: mustParseBigInt(t, "43723409723490243842378942378901237502734019231380123"), Exp: 83, Valid: true}, new(pgtype.Numeric), isExpectedEqNumeric(pgtype.Numeric{Int: mustParseBigInt(t, "43723409723490243842378942378901237502734019231380123"), Exp: 83, Valid: true})},
		{pgtype.Numeric{Int: mustParseBigInt(t, "43723409723490243842378942378901237502734019231380123"), Exp: 84, Valid: true}, new(pgtype.Numeric), isExpectedEqNumeric(pgtype.Numeric{Int: mustParseBigInt(t, "43723409723490243842378942378901237502734019231380123"), Exp: 84, Valid: true})},
		{pgtype.Numeric{Int: mustParseBigInt(t, "913423409823409243892349028349023482934092340892390101"), Exp: -14021, Valid: true}, new(pgtype.Numeric), isExpectedEqNumeric(pgtype.Numeric{Int: mustParseBigInt(t, "913423409823409243892349028349023482934092340892390101"), Exp: -14021, Valid: true})},
		{pgtype.Numeric{Int: mustParseBigInt(t, "13423409823409243892349028349023482934092340892390101"), Exp: -90, Valid: true}, new(pgtype.Numeric), isExpectedEqNumeric(pgtype.Numeric{Int: mustParseBigInt(t, "13423409823409243892349028349023482934092340892390101"), Exp: -90, Valid: true})},
		{pgtype.Numeric{Int: mustParseBigInt(t, "13423409823409243892349028349023482934092340892390101"), Exp: -91, Valid: true}, new(pgtype.Numeric), isExpectedEqNumeric(pgtype.Numeric{Int: mustParseBigInt(t, "13423409823409243892349028349023482934092340892390101"), Exp: -91, Valid: true})},
		{pgtype.Numeric{Int: mustParseBigInt(t, "13423409823409243892349028349023482934092340892390101"), Exp: -92, Valid: true}, new(pgtype.Numeric), isExpectedEqNumeric(pgtype.Numeric{Int: mustParseBigInt(t, "13423409823409243892349028349023482934092340892390101"), Exp: -92, Valid: true})},
		{pgtype.Numeric{Int: mustParseBigInt(t, "13423409823409243892349028349023482934092340892390101"), Exp: -93, Valid: true}, new(pgtype.Numeric), isExpectedEqNumeric(pgtype.Numeric{Int: mustParseBigInt(t, "13423409823409243892349028349023482934092340892390101"), Exp: -93, Valid: true})},
		{pgtype.Numeric{NaN: true, Valid: true}, new(pgtype.Numeric), isExpectedEqNumeric(pgtype.Numeric{NaN: true, Valid: true})},
		{pgtype.Numeric{InfinityModifier: pgtype.Infinity, Valid: true}, new(pgtype.Numeric), isExpectedEqNumeric(pgtype.Numeric{InfinityModifier: pgtype.Infinity, Valid: true})},
		{pgtype.Numeric{InfinityModifier: pgtype.NegativeInfinity, Valid: true}, new(pgtype.Numeric), isExpectedEqNumeric(pgtype.Numeric{InfinityModifier: pgtype.NegativeInfinity, Valid: true})},
		{longestNumeric, new(pgtype.Numeric), isExpectedEqNumeric(longestNumeric)},
		{mustParseNumeric(t, "1"), new(int64), isExpectedEq(int64(1))},
		{math.NaN(), new(float64), func(a interface{}) bool { return math.IsNaN(a.(float64)) }},
		{float32(math.NaN()), new(float32), func(a interface{}) bool { return math.IsNaN(float64(a.(float32))) }},
		{math.Inf(1), new(float64), isExpectedEq(math.Inf(1))},
		{float32(math.Inf(1)), new(float32), isExpectedEq(float32(math.Inf(1)))},
		{math.Inf(-1), new(float64), isExpectedEq(math.Inf(-1))},
		{float32(math.Inf(-1)), new(float32), isExpectedEq(float32(math.Inf(-1)))},
		{int64(-1), new(pgtype.Numeric), isExpectedEqNumeric(mustParseNumeric(t, "-1"))},
		{int64(0), new(pgtype.Numeric), isExpectedEqNumeric(mustParseNumeric(t, "0"))},
		{int64(1), new(pgtype.Numeric), isExpectedEqNumeric(mustParseNumeric(t, "1"))},
		{int64(math.MinInt64), new(pgtype.Numeric), isExpectedEqNumeric(mustParseNumeric(t, strconv.FormatInt(math.MinInt64, 10)))},
		{int64(math.MinInt64 + 1), new(pgtype.Numeric), isExpectedEqNumeric(mustParseNumeric(t, strconv.FormatInt(math.MinInt64+1, 10)))},
		{int64(math.MaxInt64), new(pgtype.Numeric), isExpectedEqNumeric(mustParseNumeric(t, strconv.FormatInt(math.MaxInt64, 10)))},
		{int64(math.MaxInt64 - 1), new(pgtype.Numeric), isExpectedEqNumeric(mustParseNumeric(t, strconv.FormatInt(math.MaxInt64-1, 10)))},
		{pgtype.Numeric{}, new(pgtype.Numeric), isExpectedEq(pgtype.Numeric{})},
		{nil, new(pgtype.Numeric), isExpectedEq(pgtype.Numeric{})},
	})
}

func TestNumericCodecFuzz(t *testing.T) {
	r := rand.New(rand.NewSource(0))
	max := &big.Int{}
	max.SetString("9999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999", 10)

	tests := make([]testutil.TranscodeTestCase, 0, 2000)
	for i := 0; i < 10; i++ {
		for j := -50; j < 50; j++ {
			num := (&big.Int{}).Rand(r, max)

			n := pgtype.Numeric{Int: num, Exp: int32(j), Valid: true}
			tests = append(tests, testutil.TranscodeTestCase{n, new(pgtype.Numeric), isExpectedEqNumeric(n)})

			negNum := &big.Int{}
			negNum.Neg(num)
			n = pgtype.Numeric{Int: negNum, Exp: int32(j), Valid: true}
			tests = append(tests, testutil.TranscodeTestCase{n, new(pgtype.Numeric), isExpectedEqNumeric(n)})
		}
	}

	testutil.RunTranscodeTests(t, "numeric", tests)
}

func TestNumericMarshalJSON(t *testing.T) {
	conn := testutil.MustConnectPgx(t)
	defer testutil.MustCloseContext(t, conn)

	for i, tt := range []struct {
		decString string
	}{
		{"NaN"},
		{"0"},
		{"1"},
		{"-1"},
		{"1000000000000000000"},
		{"1234.56789"},
		{"1.56789"},
		{"0.00000000000056789"},
		{"0.00123000"},
		{"123e-3"},
		{"243723409723490243842378942378901237502734019231380123e23790"},
		{"3409823409243892349028349023482934092340892390101e-14021"},
	} {
		var num pgtype.Numeric
		var pgJSON string
		err := conn.QueryRow(context.Background(), `select $1::numeric, to_json($1::numeric)`, tt.decString).Scan(&num, &pgJSON)
		require.NoErrorf(t, err, "%d", i)

		goJSON, err := json.Marshal(num)
		require.NoErrorf(t, err, "%d", i)

		require.Equal(t, pgJSON, string(goJSON))
	}
}
