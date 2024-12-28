package crudui

import "reflect"

// IsFieldKindSupported checks if a field kind is supported by this module
func IsFieldKindSupported(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	case reflect.String, reflect.Bool:
		return true
	case reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

func isStructField(u interface{}, field string) bool {
	v := reflect.ValueOf(u)
	i := reflect.Indirect(v)
	s := i.Type()

	for j := 0; j < s.NumField(); j++ {
		f := s.Field(j)
		k := f.Type.Kind()

		if !IsFieldKindSupported(k) {
			continue
		}

		if f.Name == field {
			return true
		}
	}

	return false
}

func getStructName(u interface{}) string {
	v := reflect.ValueOf(u)
	i := reflect.Indirect(v)
	s := i.Type()
	return s.Name()
}

func getStructFieldNames(u interface{}) []string {
	v := reflect.ValueOf(u)
	i := reflect.Indirect(v)
	s := i.Type()

	names := []string{}

	for j := 0; j < s.NumField(); j++ {
		f := s.Field(j)
		k := f.Type.Kind()

		if !IsFieldKindSupported(k) {
			continue
		}

		names = append(names, f.Name)
	}

	return names
}
