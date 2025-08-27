package db

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/jackc/pgx/v4"
	"cement/reflector"
	"cement/stringtool"
	"cement/uuid"
	"gorest/resource"
)

const TablePrefix = "gr_"

const (
	joinSqlTemplateContent string = "select {{.OwnedTable}}.* from {{.OwnedTable}} inner join {{.RelTable}} on ({{.OwnedTable}}.id={{.RelTable}}.{{.Owned}} and {{.RelTable}}.{{.Owner}}=$1)"
)

var joinSqlTemplate *template.Template

func init() {
	joinSqlTemplate, _ = template.New("").Parse(joinSqlTemplateContent)
}

func resourceTableName(typ ResourceType) string {
	return TablePrefix + string(typ)
}

func createTableSql(descriptor *ResourceDescriptor) string {
	var buf bytes.Buffer
	buf.WriteString("create table if not exists ")
	buf.WriteString(resourceTableName(descriptor.Typ))
	buf.WriteString(" (")

	for _, field := range descriptor.Fields {
		buf.WriteString(field.Name)
		buf.WriteString(" ")
		buf.WriteString(postgresqlTypeMap[field.Type])
		if field.Unique {
			buf.WriteString(" ")
			buf.WriteString("unique")
		}

		if field.Check == Positive {
			buf.WriteString(" check(")
			buf.WriteString(field.Name)
			buf.WriteString(" > 0)")
		}
		buf.WriteString(",")
	}

	for _, owner := range descriptor.Owners {
		buf.WriteString(string(owner))
		buf.WriteString(" text not null references ")
		buf.WriteString(resourceTableName(owner))
		buf.WriteString(" (id) on delete cascade")
		buf.WriteString(",")
	}

	for _, refer := range descriptor.Refers {
		buf.WriteString(string(refer))
		buf.WriteString(" text not null references ")
		buf.WriteString(resourceTableName(refer))
		buf.WriteString(" (id) on delete restrict")
		buf.WriteString(",")
	}

	if len(descriptor.Pks) > 0 {
		buf.WriteString("primary key (")
		for i, pk := range descriptor.Pks {
			if i > 0 {
				buf.WriteString(",")
			}
			buf.WriteString(string(pk))
		}
		buf.WriteString("),")
	}

	if len(descriptor.Uks) > 0 {
		buf.WriteString("unique (")
		for i, uk := range descriptor.Uks {
			if i > 0 {
				buf.WriteString(",")
			}
			buf.WriteString(string(uk))
		}
		buf.WriteString("),")
	}

	sql := (strings.TrimRight(buf.String(), ",") + ")")
	return sql
}

func insertSqlArgsAndID(meta *ResourceMeta, r resource.Resource) (string, []interface{}, error) {
	typ := ResourceDBType(r)
	descriptor, err := meta.GetDescriptor(typ)
	if err != nil {
		return "", nil, fmt.Errorf("get %v descriptor failed %v", typ, err.Error())
	}

	tableName := resourceTableName(descriptor.Typ)
	fieldCount := len(descriptor.Fields) + len(descriptor.Owners) + len(descriptor.Refers)
	markers := make([]string, 0, fieldCount)
	for i := 1; i <= fieldCount; i++ {
		markers = append(markers, "$"+strconv.Itoa(i))
	}
	sql := strings.Join([]string{"insert into", tableName, "values(", strings.Join(markers, ","), ")"}, " ")
	args := make([]interface{}, 0, fieldCount)

	id := r.GetID()
	if id == "" {
		id, _ = uuid.Gen()
		r.SetID(id)
	}

	val, isOk := reflector.GetStructFromPointer(r)
	if isOk == false {
		return "", nil, fmt.Errorf("%v is not pointer to resource", reflect.TypeOf(r).Kind().String())
	}

	for _, field := range descriptor.Fields {
		if field.Name == IDField {
			args = append(args, id)
		} else if field.Name == CreateTimeField {
			args = append(args, r.GetCreationTimestamp())
		} else {
			fieldVal := val.FieldByName(stringtool.ToUpperCamel(field.Name))
			args = append(args, fieldVal.Interface())
		}
	}

	for _, owner := range descriptor.Owners {
		args = append(args, val.FieldByName(stringtool.ToUpperCamel(string(owner))).Interface())
	}

	for _, refer := range descriptor.Refers {
		args = append(args, val.FieldByName(stringtool.ToUpperCamel(string(refer))).Interface())
	}

	return sql, args, nil
}

func selectSqlAndArgs(meta *ResourceMeta, typ ResourceType, conds map[string]interface{}) (string, []interface{}, error) {
	descriptor, err := meta.GetDescriptor(typ)
	if err != nil {
		return "", nil, fmt.Errorf("get descriptor for %v failed %v", typ, err.Error())
	}

	orderStat := "order by id"
	if order_, ok := conds["orderby"]; ok == true {
		if order, ok := order_.(string); ok == false {
			return "", nil, fmt.Errorf("order argument isn't string:%v", order_)
		} else {
			orderStat = fmt.Sprintf("order by %s", stringtool.ToSnake(order))
			delete(conds, "orderby")
		}
	}

	limitStat := ""
	if limit_, ok := conds["limit"]; ok == true {
		if offset_, ok := conds["offset"]; ok == true {
			limit, _ := limit_.(int)
			offset, _ := offset_.(int)
			delete(conds, "limit")
			delete(conds, "offset")
			limitStat = fmt.Sprintf("limit %d offset %d", limit, offset)
		}
	}

	whereState, args, err := getSqlWhereState(conds)
	if err != nil {
		return "", nil, err
	} else if whereState == "" {
		return strings.Join([]string{"select * from ", resourceTableName(descriptor.Typ), orderStat, limitStat}, " "), nil, nil
	} else {
		return strings.Join([]string{"select * from", resourceTableName(descriptor.Typ), "where", whereState, orderStat, limitStat}, " "), args, nil
	}
}

func deleteSqlAndArgs(meta *ResourceMeta, typ ResourceType, conds map[string]interface{}) (string, []interface{}, error) {
	descriptor, err := meta.GetDescriptor(typ)
	if err != nil {
		return "", nil, fmt.Errorf("get descriptor for %v failed %v", typ, err.Error())
	}

	if len(conds) == 0 {
		return ("delete from " + resourceTableName(descriptor.Typ)), nil, nil
	}

	whereState := make([]string, 0, len(conds))
	args := make([]interface{}, 0, len(conds))
	markerSeq := 1
	for k, v := range conds {
		whereState = append(whereState, stringtool.ToSnake(k)+"=$"+strconv.Itoa(markerSeq))
		args = append(args, v)
		markerSeq += 1
	}
	whereSeq := strings.Join(whereState, " and ")
	return strings.Join([]string{"delete from", resourceTableName(descriptor.Typ), "where", whereSeq}, " "), args, nil
}

//select count(*) from zc_zone where zdnsuser=$1
func existsSqlAndArgs(meta *ResourceMeta, typ ResourceType, conds map[string]interface{}) (string, []interface{}, error) {
	descriptor, err := meta.GetDescriptor(typ)
	if err != nil {
		return "", nil, fmt.Errorf("get descriptor for %v failed %v", typ, err.Error())
	}

	if len(conds) == 0 {
		return ("select (exists (select 1 from " + resourceTableName(descriptor.Typ) + " limit 1))"), nil, nil
	}

	whereState := make([]string, 0, len(conds))
	args := make([]interface{}, 0, len(conds))
	markerSeq := 1

	for k, v := range conds {
		whereState = append(whereState, stringtool.ToSnake(k)+"=$"+strconv.Itoa(markerSeq))
		args = append(args, v)
		markerSeq += 1
	}

	whereSeq := strings.Join(whereState, " and ")
	return strings.Join([]string{"select (exists (select 1 from ", resourceTableName(descriptor.Typ), "where", whereSeq, "limit 1))"}, " "), args, nil
}

//select count(*) from zc_zone where zdnsuser=$1
func countSqlAndArgs(meta *ResourceMeta, typ ResourceType, conds map[string]interface{}) (string, []interface{}, error) {
	descriptor, err := meta.GetDescriptor(typ)
	if err != nil {
		return "", nil, fmt.Errorf("get descriptor for %v failed %v", typ, err.Error())
	}

	whereState, args, err := getSqlWhereState(conds)
	if err != nil {
		return "", nil, err
	} else if whereState == "" {
		return ("select count(*) from " + resourceTableName(descriptor.Typ)), nil, nil
	} else {
		return strings.Join([]string{"select count(*) from", resourceTableName(descriptor.Typ), "where", whereState}, " "), args, nil
	}
}

//UPDATE films SET kind = 'Dramatic' WHERE kind = 'Drama';
func updateSqlAndArgs(meta *ResourceMeta, typ ResourceType, newVals map[string]interface{}, conds map[string]interface{}) (string, []interface{}, error) {
	descriptor, err := meta.GetDescriptor(typ)
	if err != nil {
		return "", nil, fmt.Errorf("get descriptor for %v failed %v", typ, err.Error())
	}

	setState := make([]string, 0, len(newVals))
	whereState := make([]string, 0, len(conds))
	args := make([]interface{}, 0, len(newVals)+len(conds))
	markerSeq := 1
	for k, v := range newVals {
		setState = append(setState, stringtool.ToSnake(k)+"=$"+strconv.Itoa(markerSeq))
		args = append(args, v)
		markerSeq += 1

	}

	for k, v := range conds {
		whereState = append(whereState, stringtool.ToSnake(k)+"=$"+strconv.Itoa(markerSeq))
		args = append(args, v)
		markerSeq += 1
	}

	setSeq := strings.Join(setState, ",")
	whereSeq := strings.Join(whereState, " and ")
	return strings.Join([]string{"update", resourceTableName(descriptor.Typ), "set", setSeq, "where", whereSeq}, " "), args, nil
}

type joinSqlParams struct {
	OwnedTable string
	RelTable   string
	Owned      string
	Owner      string
}

func joinSelectSqlAndArgs(meta *ResourceMeta, ownerTyp ResourceType, ownedTyp ResourceType, ownerID string) (string, []interface{}, error) {
	relationTyp := strings.ToLower(string(ownerTyp)) + "_" + strings.ToLower(string(ownedTyp))
	ownedDescriptor, err := meta.GetDescriptor(ownedTyp)
	if err != nil {
		return "", nil, fmt.Errorf("get descriptor for %v failed %v", ownedTyp, err.Error())
	}

	relationDescriptor, err := meta.GetDescriptor(ResourceType(relationTyp))
	if err != nil {
		return "", nil, fmt.Errorf("get descriptor for %v failed %v", relationTyp, err.Error())
	}

	params := &joinSqlParams{resourceTableName(ownedDescriptor.Typ),
		resourceTableName(relationDescriptor.Typ),
		string(ownedTyp),
		string(ownerTyp)}

	var b bytes.Buffer
	joinSqlTemplate.Execute(&b, params)
	return b.String(), []interface{}{ownerID}, nil
}

func getSqlWhereState(conds map[string]interface{}) (string, []interface{}, error) {
	if len(conds) == 0 {
		return "", nil, nil
	}

	searchKeys := []string{}
	if keys_, ok := conds["search"]; ok {
		if keys, ok := keys_.(string); ok {
			searchKeys = strings.Split(keys, ",")
		}
		delete(conds, "search")
	}

	matchListKeys := []string{}
	if keys_, ok := conds["match_list"]; ok {
		if keys, ok := keys_.(string); ok {
			matchListKeys = strings.Split(keys, ",")
		}
		delete(conds, "match_list")
	}

	whereState := make([]string, 0, len(conds))
	args := make([]interface{}, 0, len(conds))
	markerSeq := 1
	for k, v := range conds {
		isSearchKey := false
		for _, sk := range searchKeys {
			if k == sk {
				isSearchKey = true
				break
			}
		}

		isMatchListKey := false
		for _, mk := range matchListKeys {
			if k == mk {
				isMatchListKey = true
				break
			}
		}

		if isSearchKey {
			whereState = append(whereState, stringtool.ToSnake(k)+" like $"+strconv.Itoa(markerSeq))
			if sv, ok := v.(string); ok == true {
				args = append(args, "%"+sv+"%")
				markerSeq += 1
			} else {
				return "", nil, fmt.Errorf("search condition isn't string, but %v", v)
			}
		} else if isMatchListKey {
			if sv, ok := v.(string); ok == true {
				orStatSegs := []string{}
				matchList := strings.Split(sv, ",")
				for _, mv := range matchList {
					orStatSegs = append(orStatSegs, fmt.Sprintf("%s=$%d", stringtool.ToSnake(k), markerSeq))
					markerSeq += 1
					args = append(args, mv)
				}
				whereState = append(whereState, "( "+strings.Join(orStatSegs, " or ")+")")
			} else {
				return "", nil, fmt.Errorf("match condition isn't string, but %v", v)
			}
		} else {
			whereState = append(whereState, stringtool.ToSnake(k)+"=$"+strconv.Itoa(markerSeq))
			args = append(args, v)
			markerSeq += 1
		}
	}

	return strings.Join(whereState, " and "), args, nil
}

func rowsToResources(rows pgx.Rows, out interface{}) error {
	goTyp := reflect.TypeOf(out)
	if goTyp.Kind() != reflect.Ptr || goTyp.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("output isn't a pointer to slice")
	}

	slice := reflect.Indirect(reflect.ValueOf(out))
	if slice.Type().Elem().Kind() != reflect.Ptr {
		return fmt.Errorf("output isn't a pointer to slice of pointer")
	}
	typ := slice.Type().Elem().Elem()

	for rows.Next() {
		elem := reflect.New(typ)
		fd := rows.FieldDescriptions()
		fields := make([]interface{}, 0, len(fd))
		var id string
		var createTime time.Time
		for _, d := range fd {
			if string(d.Name) == IDField {
				fields = append(fields, &id)
			} else if string(d.Name) == CreateTimeField {
				fields = append(fields, &createTime)
			} else {
				fieldName := stringtool.ToUpperCamel(string(d.Name))
				fields = append(fields, elem.Elem().FieldByName(fieldName).Addr().Interface())
			}
		}
		err := rows.Scan(fields...)
		if err != nil {
			return err
		}
		r, ok := elem.Interface().(resource.Resource)
		if !ok {
			return fmt.Errorf("output isn't a pointer to slice of resource")
		}
		r.SetID(id)
		r.SetCreationTimestamp(createTime)
		slice.Set(reflect.Append(slice, elem))
	}
	return nil
}
