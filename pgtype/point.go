package pgtype

import (
	"bytes"
	"database/sql/driver"
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/jackc/pgio"
)

type Vec2 struct {
	X float64
	Y float64
}

type PointScanner interface {
	ScanPoint(v Point) error
}

type PointValuer interface {
	PointValue() (Point, error)
}

type Point struct {
	P     Vec2
	Valid bool
}

func (p *Point) ScanPoint(v Point) error {
	*p = v
	return nil
}

func (p Point) PointValue() (Point, error) {
	return p, nil
}

func parsePoint(src []byte) (*Point, error) {
	if src == nil || bytes.Compare(src, []byte("null")) == 0 {
		return &Point{}, nil
	}

	if len(src) < 5 {
		return nil, fmt.Errorf("invalid length for point: %v", len(src))
	}
	if src[0] == '"' && src[len(src)-1] == '"' {
		src = src[1 : len(src)-1]
	}
	parts := strings.SplitN(string(src[1:len(src)-1]), ",", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid format for point")
	}

	x, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return nil, err
	}

	y, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return nil, err
	}

	return &Point{P: Vec2{x, y}, Valid: true}, nil
}

// Scan implements the database/sql Scanner interface.
func (dst *Point) Scan(src interface{}) error {
	if src == nil {
		*dst = Point{}
		return nil
	}

	switch src := src.(type) {
	case string:
		return scanPlanTextAnyToPointScanner{}.Scan([]byte(src), dst)
	}

	return fmt.Errorf("cannot scan %T", src)
}

// Value implements the database/sql/driver Valuer interface.
func (src Point) Value() (driver.Value, error) {
	if !src.Valid {
		return nil, nil
	}

	buf, err := PointCodec{}.PlanEncode(nil, 0, TextFormatCode, src).Encode(src, nil)
	if err != nil {
		return nil, err
	}
	return string(buf), err
}

func (src Point) MarshalJSON() ([]byte, error) {
	if !src.Valid {
		return []byte("null"), nil
	}

	var buff bytes.Buffer
	buff.WriteByte('"')
	buff.WriteString(fmt.Sprintf("(%g,%g)", src.P.X, src.P.Y))
	buff.WriteByte('"')
	return buff.Bytes(), nil
}

func (dst *Point) UnmarshalJSON(point []byte) error {
	p, err := parsePoint(point)
	if err != nil {
		return err
	}
	*dst = *p
	return nil
}

type PointCodec struct{}

func (PointCodec) FormatSupported(format int16) bool {
	return format == TextFormatCode || format == BinaryFormatCode
}

func (PointCodec) PreferredFormat() int16 {
	return BinaryFormatCode
}

func (PointCodec) PlanEncode(ci *ConnInfo, oid uint32, format int16, value interface{}) EncodePlan {
	if _, ok := value.(PointValuer); !ok {
		return nil
	}

	switch format {
	case BinaryFormatCode:
		return encodePlanPointCodecBinary{}
	case TextFormatCode:
		return encodePlanPointCodecText{}
	}

	return nil
}

type encodePlanPointCodecBinary struct{}

func (encodePlanPointCodecBinary) Encode(value interface{}, buf []byte) (newBuf []byte, err error) {
	point, err := value.(PointValuer).PointValue()
	if err != nil {
		return nil, err
	}

	if !point.Valid {
		return nil, nil
	}

	buf = pgio.AppendUint64(buf, math.Float64bits(point.P.X))
	buf = pgio.AppendUint64(buf, math.Float64bits(point.P.Y))
	return buf, nil
}

type encodePlanPointCodecText struct{}

func (encodePlanPointCodecText) Encode(value interface{}, buf []byte) (newBuf []byte, err error) {
	point, err := value.(PointValuer).PointValue()
	if err != nil {
		return nil, err
	}

	if !point.Valid {
		return nil, nil
	}

	return append(buf, fmt.Sprintf(`(%s,%s)`,
		strconv.FormatFloat(point.P.X, 'f', -1, 64),
		strconv.FormatFloat(point.P.Y, 'f', -1, 64),
	)...), nil
}

func (PointCodec) PlanScan(ci *ConnInfo, oid uint32, format int16, target interface{}, actualTarget bool) ScanPlan {

	switch format {
	case BinaryFormatCode:
		switch target.(type) {
		case PointScanner:
			return scanPlanBinaryPointToPointScanner{}
		}
	case TextFormatCode:
		switch target.(type) {
		case PointScanner:
			return scanPlanTextAnyToPointScanner{}
		}
	}

	return nil
}

func (c PointCodec) DecodeDatabaseSQLValue(ci *ConnInfo, oid uint32, format int16, src []byte) (driver.Value, error) {
	return codecDecodeToTextFormat(c, ci, oid, format, src)
}

func (c PointCodec) DecodeValue(ci *ConnInfo, oid uint32, format int16, src []byte) (interface{}, error) {
	if src == nil {
		return nil, nil
	}

	var point Point
	err := codecScan(c, ci, oid, format, src, &point)
	if err != nil {
		return nil, err
	}
	return point, nil
}

type scanPlanBinaryPointToPointScanner struct{}

func (scanPlanBinaryPointToPointScanner) Scan(src []byte, dst interface{}) error {
	scanner := (dst).(PointScanner)

	if src == nil {
		return scanner.ScanPoint(Point{})
	}

	if len(src) != 16 {
		return fmt.Errorf("invalid length for point: %v", len(src))
	}

	x := binary.BigEndian.Uint64(src)
	y := binary.BigEndian.Uint64(src[8:])

	return scanner.ScanPoint(Point{
		P:     Vec2{math.Float64frombits(x), math.Float64frombits(y)},
		Valid: true,
	})
}

type scanPlanTextAnyToPointScanner struct{}

func (scanPlanTextAnyToPointScanner) Scan(src []byte, dst interface{}) error {
	scanner := (dst).(PointScanner)

	if src == nil {
		return scanner.ScanPoint(Point{})
	}

	if len(src) < 5 {
		return fmt.Errorf("invalid length for point: %v", len(src))
	}

	parts := strings.SplitN(string(src[1:len(src)-1]), ",", 2)
	if len(parts) < 2 {
		return fmt.Errorf("invalid format for point")
	}

	x, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return err
	}

	y, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return err
	}

	return scanner.ScanPoint(Point{P: Vec2{x, y}, Valid: true})
}
