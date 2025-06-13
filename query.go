package page

import (
	"fmt"
	"gorm.io/gorm"
	"reflect"
	"slices"
	"strings"
)

const (
	DefaultSortFields = "created_at"
)

var (
	orderMap      = map[bool]string{false: "ASC", true: "DESC"}
	comparableMap = map[bool]string{false: "<", true: ">"}
)

type Result[T any] struct {
	Data      []*T
	Total     int64
	PrevToken string
	NextToken string
}

type Paginator[T any] struct {
	DB          *gorm.DB // db提前定义好where条件
	Fields      string   // 排序字段 按照数据库字段名 e.g. "created_at,id"sql即为 "created_at DESC,id DESC"
	Start       int
	Limit       int
	Sequence    bool   // true为DESC false为ASC
	PrimaryKeys string // 主键默认为"id" e.g 复合主键"tenant_id, id"  延迟关联需要使用
}

// 默认以created_at 降序查找
func NewPaginator[T any](db *gorm.DB, start, limit int) *Paginator[T] {
	p := &Paginator[T]{
		DB:          db,
		Fields:      DefaultSortFields,
		Sequence:    true,
		Start:       start,
		Limit:       limit,
		PrimaryKeys: "id",
	}
	return p
}

func (p *Paginator[T]) SetSequence(sequence bool) *Paginator[T] {
	p.Sequence = sequence
	return p
}

// e.g. "created_at,id" 以创建时间为主排序 相同时以id为次排序
// e.g. "date" 单以日期排序
func (p *Paginator[T]) SetFields(fields string) *Paginator[T] {
	p.Fields = strings.ReplaceAll(fields, " ", "")
	return p
}

func (p *Paginator[T]) SetPrimaryKeys(keys string) *Paginator[T] {
	p.PrimaryKeys = strings.ReplaceAll(keys, " ", "")
	return p
}

func (p *Paginator[T]) Paginate(prevToken, nextToken string) (*Result[T], error) {
	var (
		res        = new(Result[T])
		orderField = getOrderFields(p.Fields, orderMap[p.Sequence])
	)

	count, err := p.optimizedCount()
	if err != nil {
		return nil, err
	} else if count == 0 {
		return res, nil
	}
	res.Total = count

	switch {
	case p.Start == 0 && prevToken == "" && nextToken == "": // 普通翻页
		p.DB = p.DB.Order(orderField).Limit(p.Limit)
	case p.Start > 0 && prevToken == "" && nextToken == "": // 延迟关联
		p.DB = p.getDelayedAssociationDB(orderField)
	default: // 游标分页
		var token string
		if prevToken != "" {
			token = prevToken
			p.Sequence = !p.Sequence
			orderField = getOrderFields(p.Fields, orderMap[p.Sequence])
		} else {
			token = nextToken
		}
		pageData, err := DecodePageToken[map[string]any](token)
		if err != nil {
			return nil, err
		}

		isGt := (nextToken != "" && !p.Sequence) || (prevToken != "" && !p.Sequence) // 上一页且倒序 || 下一页且升序
		p.DB = p.DB.Where(fmt.Sprintf("(%s) %s (%s)", p.Fields, comparableMap[isGt], getCompareValue(p.Fields, *pageData))).Order(orderField).Limit(p.Limit)
	}

	if err := p.DB.Find(&res.Data).Error; err != nil {
		return nil, err
	} else if len(res.Data) == 0 {
		return res, nil
	}

	// 上一页就把顺序换一下
	if prevToken != "" {
		slices.Reverse(res.Data)
	}

	// 生成 prevToken 和 nextToken
	start, err := getKeyValue(res.Data[0], p.Fields)
	if err != nil {
		return nil, err
	}
	res.PrevToken = EncodePageToken(start)

	end, err := getKeyValue(res.Data[len(res.Data)-1], p.Fields)
	if err != nil {
		return nil, err
	}
	res.NextToken = EncodePageToken(end)

	return res, nil
}

func (p *Paginator[T]) getDelayedAssociationDB(orderField string) *gorm.DB {
	table := p.DB.Statement.TableExpr.SQL
	fields := strings.Join(p.DB.Statement.Selects, ",")
	joins := p.DB.Statement.Joins
	// 有join 关联
	if len(joins) > 0 {
		p.DB.Statement.Joins = nil
	}
	subQuery := p.DB.Select(p.PrimaryKeys).Order(orderField).Offset(p.Start).Limit(p.Limit)
	newDB := p.DB.Session(&gorm.Session{NewDB: true}).Table(table)
	if len(joins) > 0 {
		newDB.Statement.Joins = joins
	}
	if fields != "" {
		newDB = newDB.Select(fields)
	}
	return newDB.Where(fmt.Sprintf("(%s) IN (?)", p.PrimaryKeys), subQuery).Order(orderField)
}

func (p *Paginator[T]) optimizedCount() (int64, error) {
	joins := p.DB.Statement.Joins
	if joins != nil {
		p.DB.Statement.Joins = nil
	}
	defer func() {
		p.DB.Statement.Joins = joins
	}()
	
	var total int64
	err := p.DB.Count(&total).Error
	return total, err
}

// 使用反射获取第一个和最后一个元素的 key 字段和 id
func getKeyValue(obj any, sortFields string) (res map[string]any, err error) {
	sortFields = getField(sortFields)
	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	res = make(map[string]any, 2)
	t := val.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// 拿到 json tag
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" {
			continue
		}
		// 获取排序字段的值
		// 截取掉 ",omitempty" 之类的
		tagParts := strings.Split(jsonTag, ",")
		if strings.Contains(sortFields, tagParts[0]) {
			res[tagParts[0]] = val.Field(i).Interface()
		}
	}

	return
}

func getOrderFields(s string, sequence string) string {
	var res []string
	for _, i := range strings.Split(s, ",") {
		res = append(res, fmt.Sprintf("%s %s", i, sequence))
	}
	return strings.Join(res, ",")
}

func getCompareValue(s string, mapp map[string]any) string {
	formatValue := func(v any) string {
		switch v.(type) {
		case string:
			return fmt.Sprintf("'%s'", v)
		default:
			return fmt.Sprintf("%v", v)
		}
	}

	if len(mapp) == 1 {
		return formatValue(mapp[getField(s)])
	}

	fields := strings.Split(s, ",")
	return fmt.Sprintf("%s, %s", formatValue(mapp[getField(fields[0])]), formatValue(mapp[getField(fields[1])]))
}

func getField(s string) string {
	temp := strings.Split(s, ".")
	if len(temp) >= 2 {
		return temp[1]
	}
	return s
}
