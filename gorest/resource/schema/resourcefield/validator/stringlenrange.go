package validator

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"gorest/util"
)

const minLenPrefix = "minLen="
const maxLenPrefix = "maxLen="

type stringLenRangeValidator struct {
	minLen *int64
	maxLen *int64
}

type stringLenRangeValidatorBuilder struct{}

func newStringLenRangeValidator(minLen, maxLen *int64) Validator {
	return &stringLenRangeValidator{
		minLen: minLen,
		maxLen: maxLen,
	}
}

func (v *stringLenRangeValidator) Validate(val interface{}) error {
	value := reflect.ValueOf(val)
	kind := util.Inspect(value.Type())
	if kind != util.String {
		return fmt.Errorf("stringLen apply to non-string type: %v", kind)
	}
	return v.validateStringLen(value.String())
}

func (v *stringLenRangeValidator) validateStringLen(s string) error {
	l := int64(len(s))
	if v.minLen != nil && l < *v.minLen {
		return fmt.Errorf("exceed the range limit, (string len %v should >= %v)", l, *v.minLen)
	}
	if v.maxLen != nil && l >= *v.maxLen {
		return fmt.Errorf("exceed the range limit, (string len %v should < %v)", l, *v.maxLen)
	}
	return nil
}

func (b *stringLenRangeValidatorBuilder) FromTags(tags []string) (Validator, error) {
	var minLenStr, maxLenStr string
	for _, tag := range tags {
		if strings.HasPrefix(tag, minLenPrefix) {
			if minLenStr != "" {
				return nil, fmt.Errorf("string len range has duplicate min tag")
			}
			minLenStr = strings.TrimPrefix(tag, minLenPrefix)
		} else if strings.HasPrefix(tag, maxLenPrefix) {
			if maxLenStr != "" {
				return nil, fmt.Errorf("string len range has duplicate max tag")
			}
			maxLenStr = strings.TrimPrefix(tag, maxLenPrefix)
		}
	}

	if minLenStr == "" && maxLenStr == "" {
		return nil, nil
	}

	var minLen, maxLen *int64
	if minLenStr != "" {
		min, err := strconv.ParseInt(minLenStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("min value isn't valid int:%s", err.Error())
		}
		minLen = &min
	}

	if maxLenStr != "" {
		max, err := strconv.ParseInt(maxLenStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("max value isn't valid int:%s", err.Error())
		}
		maxLen = &max
	}

	if minLen != nil && maxLen != nil && *minLen >= *maxLen {
		return nil, fmt.Errorf("min value should smaller than max")
	}
	return newStringLenRangeValidator(minLen, maxLen), nil
}

func (b *stringLenRangeValidatorBuilder) SupportKind(kind util.Kind) bool {
	return kind == util.String ||
		kind == util.StringSlice ||
		kind == util.StringStringMap
}
