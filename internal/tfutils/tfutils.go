package tfutils

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	visitCounter int64
	specialNames = map[string]string{
		"alarmname":     "alarm-name",
		"labelselector": "label-selector",
	}
	ignoreCaseNames = map[string]bool{
		"annotations": true,
		"labels":      true,
	}
	ignoreCaseVisitor = map[string]bool{}
)

func newVisitID(prefix string) string {
	return prefix + "-" + strconv.FormatInt(atomic.AddInt64(&visitCounter, 1), 10)
}

// SnakeToCamel converts a snake_case string to camelCase
// |-----------------------------|
// |           Examples          |
// |--------------|--------------|
// | Input string | Return value |
// |--------------|--------------|
// | "api_ver_1"  | "apiVer1"    |
// | "__lag"      | "lag"        |
// | "_members"   | "members"    |
// |--------------|--------------|
func SnakeToCamel(str string) string {
	parts := strings.Split(str, "_")
	for i := 1; i < len(parts); i++ {
		// If the first part is empty, skip capitalizing the 2nd part, e.g. "_members"
		if parts[0] != "" && len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

// CamelToSnake converts a camelCase string to snake_case
// |--------------------------------|
// |           Examples             |
// |---------------|----------------|
// | Input string  | Return value   |
// |---------------|----------------|
// | "apiVersion1" | "api_version1" |
// | "__lag"       | "__lag"        |
// | "_MemberS"    | "_member_s"    |
// |---------------|----------------|
func CamelToSnake(str string) string {
	// Insert an underscore before all capital letters that are not at the start
	re := regexp.MustCompile(`([a-z0-9])([A-Z])`)
	snake := re.ReplaceAllString(str, "${1}_${2}")
	return strings.ToLower(snake)
}

func newObjectTypableNull(ctx context.Context, objTypable basetypes.ObjectTypable) (attr.Value, error) {
	if objTypable == nil {
		return nil, errors.New("value is nil")
	}
	// In order to determine the attribute types of the object, we need to
	// first get the object value from the object valuable. This is because
	// the object valuable may be a custom type that implements the ObjectTypable
	// interface, and we need to get the attribute types from the object value.
	objVal, d := objTypable.ValueType(ctx).(basetypes.ObjectValuable).ToObjectValue(ctx)
	if d.HasError() {
		return nil, fmt.Errorf("failed to get obj value from obj valuable: %v", d)
	}

	// Create a new object null value with the attribute types of the object
	objNull := types.ObjectNull(objVal.AttributeTypes(ctx))
	tflog.Debug(ctx, "newObjectTypableNull()", map[string]any{
		"objNull":   objNull.String(),
		"isNull":    objNull.IsNull(),
		"isUnknown": objNull.IsUnknown(),
	})
	// Convert the object null value to a terraform null value
	// and then convert it back to the object value from the terraform null.
	// This is needed because the object null value may be a custom type
	// that implements the ObjectTypable interface, and we need to get the
	// custom type's null value rather than the generic object null value.
	tfNull, err := objNull.ToTerraformValue(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tf value from obj: %v", err)
	}
	nullValue, err := objTypable.ValueFromTerraform(ctx, tfNull)
	if err != nil {
		return nil, fmt.Errorf("failed to get value from obj: %v", err)
	}
	tflog.Debug(ctx, "newObjectTypableNull()", map[string]any{
		"attrType":  objTypable.String(),
		"nullValue": nullValue.String(),
		"isNull":    nullValue.IsNull(),
		"isUnknown": nullValue.IsUnknown(),
	})
	return nullValue, nil
}

// Takes a context and an attr.Value and returns a null value corresponding to the value's type.
func newNullValue(ctx context.Context, attrValIf attr.Value) (attr.Value, error) {
	if attrValIf == nil {
		return nil, errors.New("value is nil")
	}
	switch attrType := attrValIf.Type(ctx).(type) {
	case basetypes.BoolType:
		return types.BoolNull(), nil
	case basetypes.DynamicType:
		return types.DynamicNull(), nil
	case basetypes.Float32Type:
		return types.Float32Null(), nil
	case basetypes.Float64Type:
		return types.Float64Null(), nil
	case basetypes.Int32Type:
		return types.Int32Null(), nil
	case basetypes.Int64Type:
		return types.Int64Null(), nil
	case basetypes.ListType:
		return types.ListNull(attrType.ElemType), nil
	case basetypes.MapType:
		return types.MapNull(attrType.ElemType), nil
	case basetypes.NumberType:
		return types.NumberNull(), nil
	case basetypes.ObjectType:
		return types.ObjectNull(attrType.AttrTypes), nil
	case basetypes.SetType:
		return types.SetNull(attrType.ElemType), nil
	case basetypes.StringType:
		return types.StringNull(), nil
	case basetypes.ObjectTypable:
		return newObjectTypableNull(ctx, attrType)
	default:
		return nil, fmt.Errorf("unsupported type %s", attrType.String())
	}
}

// Converts any numeric value to int64. It supports various types including int, uint, string, float32, and float64.
// This is required during parsing API responses in json which return numbers as float64, to a terraform basetype.
func NumToInt64(value any) (int64, error) {
	switch v := value.(type) {
	case int:
		return int64(v), nil
	case int8:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case uint:
		return int64(v), nil
	case uint8:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint64:
		return int64(v), nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	case float32:
		return int64(v), nil
	case float64:
		return int64(v), nil
	default:
		return 0, fmt.Errorf("unsupported type: %T", value)
	}
}

// Creates a new attr.Value from the given attr.Type and any value.
// If val is nil, it returns a null value of the corresponding attr.Type.
func newValue(ctx context.Context, attrTypeIf attr.Type, val any, visitId string) (attr.Value, error) {
	if attrTypeIf == nil {
		return nil, errors.New("attr type is nil")
	}
	switch attrType := attrTypeIf.(type) {
	case basetypes.BoolType:
		if val == nil {
			return types.BoolNull(), nil
		}
		boolVal, ok := val.(bool)
		if !ok {
			return nil, fmt.Errorf("expected bool, got %T", val)
		}
		return types.BoolValue(boolVal), nil
	case basetypes.DynamicType:
		if val == nil {
			return types.DynamicNull(), nil
		}
		attrVal, ok := val.(attr.Value)
		if !ok {
			return nil, fmt.Errorf("expected attr.Value, got %T", val)
		}
		return types.DynamicValue(attrVal), nil
	case basetypes.Float32Type:
		if val == nil {
			return types.Float32Null(), nil
		}
		float32Val, ok := val.(float32)
		if !ok {
			return nil, fmt.Errorf("expected float32, got %T", val)
		}
		return types.Float32Value(float32Val), nil
	case basetypes.Float64Type:
		if val == nil {
			return types.Float64Null(), nil
		}
		float64Val, ok := val.(float64)
		if !ok {
			return nil, fmt.Errorf("expected float64, got %T", val)
		}
		return types.Float64Value(float64Val), nil
	case basetypes.Int32Type:
		if val == nil {
			return types.Int32Null(), nil
		}
		int32Val, ok := val.(int32)
		if !ok {
			return nil, fmt.Errorf("expected int32, got %T", val)
		}
		return types.Int32Value(int32Val), nil
	case basetypes.Int64Type:
		if val == nil {
			return types.Int64Null(), nil
		}
		int64Val, err := NumToInt64(val)
		if err != nil {
			return nil, fmt.Errorf("expected int64, got %T", val)
		}
		return types.Int64Value(int64Val), nil
	case basetypes.ListType:
		if val == nil {
			return types.ListNull(attrType.ElemType), nil
		}
		valuesList, ok := val.([]any)
		if !ok {
			return nil, fmt.Errorf("expected []any, got %T", val)
		}
		var newValList = make([]attr.Value, 0)
		for _, v := range valuesList {
			newVal, err := newValue(ctx, attrType.ElementType(), v, visitId)
			if err != nil {
				return nil, err
			}
			newValList = append(newValList, newVal)
		}
		listVal, d := types.ListValue(attrType.ElemType, newValList)
		if d.HasError() {
			return nil, fmt.Errorf("failed to create list value: %v", d)
		}
		return listVal, nil
	case basetypes.MapType:
		if val == nil {
			return types.MapNull(attrType.ElemType), nil
		}
		valuesMap, ok := val.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("expected map[string]any, got %T", val)
		}
		tflog.Trace(ctx, "newValue()::MapType case",
			map[string]any{"valuesMap": spew.Sdump(valuesMap), "visitId": visitId})

		newValMap := make(map[string]attr.Value)
		oldVisitId := visitId
		for k, v := range valuesMap {
			if !ignoreCaseVisitor[visitId] && ignoreCaseNames[k] {
				visitId = newVisitID(k)
				ignoreCaseVisitor[visitId] = true
			}
			tflog.Trace(ctx, "newValue()::MapType case: Processing valuesMap",
				map[string]any{"name": k, "visitId": visitId})

			newVal, err := newValue(ctx, attrType.ElementType(), v, visitId)
			if err != nil {
				return nil, err
			}
			if ignoreCaseVisitor[visitId] {
				newValMap[k] = newVal
			} else {
				newValMap[SnakeToCamel(k)] = newVal
			}
			if visitId != oldVisitId {
				tflog.Trace(ctx, "newValue()::MapType case: Deleting visitId",
					map[string]any{"name": k, "oldVisitId": oldVisitId, "newVisitId": visitId})

				delete(ignoreCaseVisitor, visitId)
				visitId = oldVisitId
			}
		}
		tflog.Trace(ctx, "newValue()::MapType case: Constructing MapValue",
			map[string]any{"newValMap": spew.Sdump(newValMap), "visitId": visitId})

		mapVal, d := types.MapValue(attrType.ElemType, newValMap)
		if d.HasError() {
			return nil, fmt.Errorf("failed to create map value from: %s, diag: %v", spew.Sdump(newValMap), d)
		}
		return mapVal, nil
	case basetypes.NumberType:
		if val == nil {
			return types.NumberNull(), nil
		}
		numVal, ok := val.(*big.Float)
		if !ok {
			return nil, fmt.Errorf("expected *big.Float, got %T", val)
		}
		return types.NumberValue(numVal), nil
	case basetypes.ObjectType:
		if val == nil {
			return types.ObjectNull(attrType.AttributeTypes()), nil
		}
		valuesMap, ok := val.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("expected map[string]any, got %T", val)
		}
		tflog.Trace(ctx, "newValue()::ObjectType case",
			map[string]any{"valuesMap": spew.Sdump(valuesMap), "visitId": visitId})

		newValMap := make(map[string]attr.Value)
		oldVisitId := visitId
		// Iterate over all the attributes of the object
		for name, aType := range attrType.AttributeTypes() {
			if !ignoreCaseVisitor[visitId] && ignoreCaseNames[name] {
				visitId = newVisitID(name)
				ignoreCaseVisitor[visitId] = true
			}
			tflog.Trace(ctx, "newValue()::ObjectType case: Processing attributes",
				map[string]any{"attrName": name, "visitId": visitId})

			newVal, err := newValue(ctx, aType, valuesMap[SnakeToCamel(name)], visitId)
			if err != nil {
				return nil, err
			}
			newValMap[name] = newVal
			if visitId != oldVisitId {
				tflog.Trace(ctx, "newValue()::ObjectType case: Deleting visitId",
					map[string]any{"attrName": name, "oldVisitId": oldVisitId, "newVisitId": visitId})

				delete(ignoreCaseVisitor, visitId)
				visitId = oldVisitId
			}
		}
		tflog.Trace(ctx, "newValue()::ObjectType case: Constructing ObjectValue",
			map[string]any{"newValMap": spew.Sdump(newValMap), "visitId": visitId})

		objVal, d := types.ObjectValue(attrType.AttributeTypes(), newValMap)
		if d.HasError() {
			return nil, fmt.Errorf("failed to create value from obj using map: %s, diag: %v", spew.Sdump(newValMap), d)
		}
		return objVal, nil
	case basetypes.SetType:
		if val == nil {
			return types.SetNull(attrType.ElemType), nil
		}
		valuesList, ok := val.([]any)
		if !ok {
			return nil, fmt.Errorf("expected []any, got %T", val)
		}
		var newValList = make([]attr.Value, 0)
		for _, v := range valuesList {
			newVal, err := newValue(ctx, attrType.ElementType(), v, visitId)
			if err != nil {
				return nil, err
			}
			newValList = append(newValList, newVal)
		}
		setVal, d := types.SetValue(attrType.ElementType(), newValList)
		if d.HasError() {
			return nil, fmt.Errorf("failed to create set value: %v", d)
		}
		return setVal, nil
	case basetypes.StringType:
		if val == nil {
			return types.StringNull(), nil
		}
		strVal, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", val)
		}
		return types.StringValue(strVal), nil
	case basetypes.ObjectTypable:
		objVal, d := attrType.ValueType(ctx).(basetypes.ObjectValuable).ToObjectValue(ctx)
		if d.HasError() {
			return nil, fmt.Errorf("failed to get obj value from obj valuable: %v", d)
		}
		if val == nil {
			return newObjectTypableNull(ctx, attrType)
		}
		valuesMap, ok := val.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("expected map[string]any, got %T", val)
		}
		tflog.Trace(ctx, "newValue()::ObjectTypable case",
			map[string]any{"valuesMap": spew.Sdump(valuesMap), "visitId": visitId})

		newValMap := make(map[string]attr.Value)
		oldVisitId := visitId
		for name, aType := range objVal.AttributeTypes(ctx) {
			if !ignoreCaseVisitor[visitId] && ignoreCaseNames[name] {
				visitId = newVisitID(name)
				ignoreCaseVisitor[visitId] = true
			}
			tflog.Trace(ctx, "newValue()::ObjectTypable case: Processing attributes",
				map[string]any{"attrName": name, "visitId": visitId})

			newVal, err := newValue(ctx, aType, valuesMap[SnakeToCamel(name)], visitId)
			if err != nil {
				return nil, err
			}
			newValMap[name] = newVal
			if visitId != oldVisitId {
				tflog.Trace(ctx, "newValue()::ObjectTypable case: Deleting visitId",
					map[string]any{"attrName": name, "oldVisitId": oldVisitId, "newVisitId": visitId})

				delete(ignoreCaseVisitor, visitId)
				visitId = oldVisitId
			}
		}
		tflog.Trace(ctx, "newValue()::ObjectTypable case: Constructing ObjectValue",
			map[string]any{"newValMap": spew.Sdump(newValMap), "visitId": visitId})

		newObjVal, d := types.ObjectValue(objVal.AttributeTypes(ctx), newValMap)
		if d.HasError() {
			return nil, fmt.Errorf("failed to create new obj from value using map: %s, diag: %v", spew.Sdump(newValMap), d)
		}
		newValue, d := attrType.ValueFromObject(ctx, newObjVal)
		if d.HasError() {
			return nil, fmt.Errorf("failed to create new value from obj: %s, diag: %v", spew.Sdump(newObjVal), d)
		}
		return newValue, nil
	default:
		return nil, fmt.Errorf("unsupported type %s", attrTypeIf.String())
	}
}

func fromValue(ctx context.Context, attrValIf attr.Value, visitId string) (any, error) {
	if attrValIf == nil {
		return nil, errors.New("value is nil")
	}
	switch attrVal := attrValIf.(type) {
	case basetypes.BoolValue:
		return attrVal.ValueBool(), nil
	case basetypes.DynamicValue:
		return fromValue(ctx, attrVal.UnderlyingValue(), visitId)
	case basetypes.Float32Value:
		return attrVal.ValueFloat32(), nil
	case basetypes.Float64Value:
		return attrVal.ValueFloat64(), nil
	case basetypes.Int32Value:
		return attrVal.ValueInt32(), nil
	case basetypes.Int64Value:
		return attrVal.ValueInt64(), nil
	case basetypes.ListValue:
		value := []any{}
		for _, v := range attrVal.Elements() {
			if v.IsNull() || v.IsUnknown() {
				continue
			}
			val, err := fromValue(ctx, v, visitId)
			if err != nil {
				return nil, err
			}
			value = append(value, val)
		}
		return value, nil
	case basetypes.MapValue:
		value := make(map[string]any)
		oldVisitId := visitId
		for k, v := range attrVal.Elements() {
			tflog.Trace(ctx, "fromValue()::Processing map elements",
				map[string]any{"name": k, "visitId": visitId})
			if v.IsNull() || v.IsUnknown() {
				continue
			}
			// If not already ignoring case, and we encounter a new attribute
			// for which we need to ignore case, generate a new visitId, and
			// start ignoring case for further visits.
			if !ignoreCaseVisitor[visitId] && ignoreCaseNames[k] {
				visitId = newVisitID(k)
				ignoreCaseVisitor[visitId] = true
			}
			val, err := fromValue(ctx, v, visitId)
			if err != nil {
				return nil, err
			}
			if ignoreCaseVisitor[visitId] {
				value[k] = val
			} else {
				value[SnakeToCamel(k)] = val
			}
			if visitId != oldVisitId {
				tflog.Trace(ctx, "fromValue()::Deleting visitId in MapValue case",
					map[string]any{"name": k, "oldVisitId": oldVisitId, "newVisitId": visitId})
				delete(ignoreCaseVisitor, visitId)
				visitId = oldVisitId
			}
		}
		tflog.Trace(ctx, "fromValue()::Returning map from MapValue case",
			map[string]any{"values": spew.Sdump(value), "visitId": visitId})
		return value, nil
	case basetypes.NumberValue:
		if attrVal.ValueBigFloat().IsInt() {
			intVal, _ := attrVal.ValueBigFloat().Int64()
			return intVal, nil
		}
		floatVal, _ := attrVal.ValueBigFloat().Float64()
		return floatVal, nil
	case basetypes.ObjectValue:
		value := make(map[string]any)
		oldVisitId := visitId
		for k, v := range attrVal.Attributes() {
			tflog.Trace(ctx, "fromValue()::Processing ObjectValue attributes",
				map[string]any{"attrName": k, "visitId": visitId})
			if v.IsNull() || v.IsUnknown() {
				continue
			}
			// If not already ignoring case, and we encounter a new attribute
			// for which we need to ignore case, generate a new visitId, and
			// start ignoring case for further visits.
			if !ignoreCaseVisitor[visitId] && ignoreCaseNames[k] {
				visitId = newVisitID(k)
				ignoreCaseVisitor[visitId] = true
			}
			val, err := fromValue(ctx, v, visitId)
			if err != nil {
				return nil, err
			}
			if ignoreCaseVisitor[visitId] {
				value[k] = val
			} else {
				value[SnakeToCamel(k)] = val
			}
			if visitId != oldVisitId {
				tflog.Trace(ctx, "fromValue()::Deleting visitId in ObjectValue case",
					map[string]any{"attrName": k, "oldVisitId": oldVisitId, "newVisitId": visitId})
				delete(ignoreCaseVisitor, visitId)
				visitId = oldVisitId
			}
		}
		tflog.Trace(ctx, "fromValue()::Returning map from ObjectValue case",
			map[string]any{"values": spew.Sdump(value), "visitId": visitId})
		return value, nil
	case basetypes.SetValue:
		value := []any{}
		for _, v := range attrVal.Elements() {
			if v.IsNull() || v.IsUnknown() {
				continue
			}
			val, err := fromValue(ctx, v, visitId)
			if err != nil {
				return nil, err
			}
			value = append(value, val)
		}
		return value, nil
	case basetypes.StringValue:
		return attrVal.ValueString(), nil
	case basetypes.TupleValue:
		value := []any{}
		for _, v := range attrVal.Elements() {
			if v.IsNull() || v.IsUnknown() {
				continue
			}
			val, err := fromValue(ctx, v, visitId)
			if err != nil {
				return nil, err
			}
			value = append(value, val)
		}
		return value, nil
	case basetypes.ObjectValuable:
		obj, d := attrVal.ToObjectValue(ctx)
		if d.HasError() {
			return nil, fmt.Errorf("failed to get obj value: %v", d)
		}
		return fromValue(ctx, obj, visitId)
	default:
		return nil, fmt.Errorf("unsupported type %s", attrValIf.Type(ctx).String())
	}
}

// Takes a context and an object value, and fills in any missing values in the
// object with null values. If an attribute value of the object is in turn an object,
// it recursively iteratives over that object and fills in any missing values.
// It returns a new root level object value with the recursively filled null values.
func fillObjectNull(ctx context.Context, objValIf basetypes.ObjectValuable) (newObj basetypes.ObjectValuable, err error) {
	if objValIf == nil {
		return nil, errors.New("value is nil")
	}
	var d diag.Diagnostics
	var objVal basetypes.ObjectValue

	// ObjectTypable interface is implemented by both basetypes.ObjectType and
	// the model's custom object types like MetadataType, SpecType, etc.
	objType := objValIf.Type(ctx).(basetypes.ObjectTypable)
	objVal, d = objValIf.ToObjectValue(ctx)
	if d.HasError() {
		return nil, fmt.Errorf("failed to get obj value: %v", d)
	}
	attrs := map[string]attr.Value{}
	// Iterate over all the attributes of the object
	for name, atVal := range objVal.Attributes() {
		tflog.Trace(ctx, "fillObjectNull()::Iterating over attrs",
			map[string]any{
				"name":      name,
				"isUnknown": atVal.IsUnknown(),
				"isNull":    atVal.IsNull(),
				"attrValue": atVal.String(),
				"attrType":  atVal.Type(ctx),
			})

		if atVal.IsUnknown() || atVal.IsNull() {
			// If the attribute value is unknown, set it to a null value
			nullValue, err := newNullValue(ctx, atVal)
			if err != nil {
				return nil, err
			}
			attrs[name] = nullValue
		} else {
			// Check if the attribute value is again an Object (that implements ObjectValuable)
			// and set the appropriate null value by recursing over that object.
			// If the attribute value is not an object, just set it to the same value
			switch atVal.(type) {
			case basetypes.ObjectValuable:
				tflog.Trace(ctx, "fillObjectNull()::ObjectValuable case", map[string]any{"name": name})
				var err error
				attrs[name], err = fillObjectNull(ctx, atVal.(basetypes.ObjectValue))
				if err != nil {
					return nil, err
				}
			default:
				tflog.Trace(ctx, "fillObjectNull()::default case", map[string]any{"name": name})
				attrs[name] = atVal
			}
		}
	}
	// Now that we have all the attributes with unknowns filled with null values, create a new object value
	newObj, d = types.ObjectValue(objVal.AttributeTypes(ctx), attrs)
	if d.HasError() {
		return nil, fmt.Errorf("failed to build obj value: %v", d)
	}
	// If objType is not an ObjectType, but a model type that implements ObjectValuable,
	// convert the newObj to the model type. This is needed because the ObjectValue is
	// a generic type and we need to convert it to the specific model type
	if _, ok := objType.(basetypes.ObjectType); !ok {
		newObj, d = objType.ValueFromObject(ctx, newObj.(basetypes.ObjectValue))
		if d.HasError() {
			return nil, fmt.Errorf("failed to get model value from obj value: %v", d)
		}
	}
	return newObj, nil
}

// Takes a context and a pointer to any model, and fills in any missing values
func FillMissingValues(ctx context.Context, model any) error {
	modelType := reflect.TypeOf(model)
	modelVal := reflect.ValueOf(model)
	tflog.Debug(ctx, "FillMissingValues()", map[string]any{
		"type": modelType.String(),
		"kind": modelType.Kind().String(),
	})

	// Check if the type is a pointer to a struct
	if modelType.Kind() != reflect.Pointer {
		return fmt.Errorf("expected pointer to struct, got %s", modelType.Kind())
	}

	attrValIf := reflect.TypeOf((*attr.Value)(nil)).Elem()
	// Iterate over all the fields of the struct
	for i := range modelType.Elem().NumField() {
		field := modelType.Elem().Field(i)
		// Check if the model struct field implements attr.Value
		if !field.Type.Implements(attrValIf) {
			tflog.Debug(ctx, fmt.Sprintf("FillMissingValues()::%s.%s does not implement attr.Value",
				modelType.Elem().String(), field.Name))
			continue
		}

		// Get the attr.Value interface from the field
		fieldVal := modelVal.Elem().Field(i)
		attrVal := fieldVal.Interface().(attr.Value)
		tflog.Debug(ctx, "FillMissingValues()::Iterating over fields", map[string]any{
			"fieldName": field.Name,
			"fieldType": field.Type.String(),
			"fieldKind": field.Type.Kind().String(),
			"isUnknown": attrVal.IsUnknown(),
			"attrVal":   attrVal.String(),
		})

		if attrVal.IsUnknown() {
			// If the attr.Value is unknown, set it to a null value
			nullValue, err := newNullValue(ctx, attrVal)
			if err != nil {
				return err
			}
			fieldVal.Set(reflect.ValueOf(nullValue))
		} else {
			// Check if the attr.Type of the field is an ObjectTypable
			// and set the appropriate null value in the field
			switch attrVal.Type(ctx).(type) {
			case basetypes.ObjectTypable:
				tflog.Trace(ctx, "FillMissingValues()::ObjectTypable case",
					map[string]any{"fieldName": field.Name, "attrVal": attrVal.String()})
				objVal, err := fillObjectNull(ctx, attrVal.(basetypes.ObjectValuable))
				if err != nil {
					return err
				}
				fieldVal.Set(reflect.ValueOf(objVal))
			}
		}
	}
	return nil
}

func StringValue(attrValIf attr.Value) string {
	if attrValIf == nil {
		return "null"
	}
	switch attrVal := attrValIf.(type) {
	case basetypes.BoolValue:
		return fmt.Sprintf("%t", attrVal.ValueBool())
	case basetypes.DynamicValue:
		return StringValue(attrVal.UnderlyingValue())
	case basetypes.Float32Value:
		return fmt.Sprintf("%f", attrVal.ValueFloat32())
	case basetypes.Float64Value:
		return fmt.Sprintf("%f", attrVal.ValueFloat64())
	case basetypes.Int32Value:
		return fmt.Sprintf("%d", attrVal.ValueInt32())
	case basetypes.Int64Value:
		return fmt.Sprintf("%d", attrVal.ValueInt64())
	case basetypes.NumberValue:
		if attrVal.ValueBigFloat().IsInt() {
			intVal, _ := attrVal.ValueBigFloat().Int64()
			return fmt.Sprintf("%d", intVal)
		}
		floatVal, _ := attrVal.ValueBigFloat().Float64()
		return fmt.Sprintf("%f", floatVal)
	case basetypes.StringValue:
		return attrVal.ValueString()
	default:
		return ""
	}
}

func ModelToStringMap(ctx context.Context, model any) (map[string]string, error) {
	body := map[string]string{}
	typ := reflect.TypeOf(model)
	val := reflect.ValueOf(model)
	tflog.Debug(ctx, "ModelToAnyMap()", map[string]any{
		"type": typ.String(),
		"kind": typ.Kind().String(),
	})

	// Check if the type is a pointer to a struct
	if typ.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("expected pointer to struct, got %s", typ.Kind())
	}

	attrValIf := reflect.TypeOf((*attr.Value)(nil)).Elem()
	for i := range typ.Elem().NumField() {
		field := typ.Elem().Field(i)
		// Check if the model struct field implements attr.Value
		if !field.Type.Implements(attrValIf) {
			tflog.Debug(ctx, fmt.Sprintf("ModelToAnyMap()::%s.%s does not implement attr.Value",
				typ.Elem().String(), field.Name))
			continue
		}
		// Compute the field name in camel case
		fieldName := strings.ToLower(field.Name[0:1]) + field.Name[1:]
		if newName, exists := specialNames[fieldName]; exists {
			fieldName = newName
		}
		attrVal := val.Elem().Field(i).Interface().(attr.Value)

		tflog.Debug(ctx, "ModelToAnyMap()::Iterating over fields", map[string]any{
			"fieldName": fieldName,
			"fieldType": field.Type.String(),
			"fieldKind": field.Type.Kind().String(),
			"isUnknown": attrVal.IsUnknown(),
			"attrVal":   attrVal.String(),
		})

		// If the attr.Value is not null and not unknown, and is a string type, use it to build the map
		if !attrVal.IsNull() && !attrVal.IsUnknown() && attrVal.Type(ctx).Equal(types.StringType) {
			// Convert the attr.Value to an appropriate Go type
			anyVal, err := fromValue(ctx, attrVal, "")
			if err != nil {
				return nil, err
			}
			if strVal, ok := anyVal.(string); ok {
				body[fieldName] = strVal
			}
		}
	}
	return body, nil
}

func ModelToAnyMap(ctx context.Context, model any) (map[string]any, error) {
	body := map[string]any{}
	typ := reflect.TypeOf(model)
	val := reflect.ValueOf(model)
	tflog.Debug(ctx, "ModelToAnyMap()", map[string]any{
		"type": typ.String(),
		"kind": typ.Kind().String(),
	})

	// Check if the type is a pointer to a struct
	if typ.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("expected pointer to struct, got %s", typ.Kind())
	}

	attrValIf := reflect.TypeOf((*attr.Value)(nil)).Elem()
	for i := range typ.Elem().NumField() {
		field := typ.Elem().Field(i)
		// Check if the model struct field implements attr.Value
		if !field.Type.Implements(attrValIf) {
			tflog.Debug(ctx, fmt.Sprintf("ModelToAnyMap()::%s.%s does not implement attr.Value",
				typ.Elem().String(), field.Name))
			continue
		}
		// Compute the field name in camel case
		fieldName := strings.ToLower(field.Name[0:1]) + field.Name[1:]
		attrVal := val.Elem().Field(i).Interface().(attr.Value)

		tflog.Debug(ctx, "ModelToAnyMap()::Iterating over fields", map[string]any{
			"fieldName": fieldName,
			"fieldType": field.Type.String(),
			"fieldKind": field.Type.Kind().String(),
			"isUnknown": attrVal.IsUnknown(),
			"attrVal":   attrVal.String(),
		})

		// If the attr.Value is not null and not unknown, use it to build the request
		if !attrVal.IsNull() && !attrVal.IsUnknown() {
			// Convert the attr.Value to an appropriate Go type
			anyVal, err := fromValue(ctx, attrVal, "")
			if err != nil {
				return nil, err
			}
			body[fieldName] = anyVal
		}
	}
	return body, nil
}

func AnyMapToModel(ctx context.Context, resp map[string]any, model any) error {
	modelType := reflect.TypeOf(model)
	modelValue := reflect.ValueOf(model)
	tflog.Debug(ctx, "AnyMapToModel()", map[string]any{
		"type": modelType.String(),
		"kind": modelType.Kind().String(),
	})

	// Check if the type is a pointer to a struct
	if modelType.Kind() != reflect.Ptr {
		return fmt.Errorf("expected pointer to struct, got %s", modelType.Kind())
	}

	attrValIf := reflect.TypeOf((*attr.Value)(nil)).Elem()
	for i := range modelType.Elem().NumField() {
		field := modelType.Elem().Field(i)
		// Check if the model struct field implements attr.Value
		if !field.Type.Implements(attrValIf) {
			tflog.Debug(ctx, fmt.Sprintf("AnyMapToModel()::%s.%s does not implement attr.Value",
				modelType.Elem().String(), field.Name))
			continue
		}
		// Compute the field name in camel case
		fieldName := strings.ToLower(field.Name[0:1]) + field.Name[1:]
		attrVal := modelValue.Elem().Field(i).Interface().(attr.Value)

		tflog.Debug(ctx, "AnyMapToModel()::Iterating over fields", map[string]any{
			"fieldName": fieldName,
			"fieldType": field.Type.String(),
			"fieldKind": field.Type.Kind().String(),
			"isUnknown": attrVal.IsUnknown(),
			"attrVal":   attrVal.String(),
		})

		newVal, err := newValue(ctx, attrVal.Type(ctx), resp[fieldName], "")
		if err != nil {
			return err
		}
		// Set the new value to the model field
		modelValue.Elem().Field(i).Set(reflect.ValueOf(newVal))
	}
	return nil
}
