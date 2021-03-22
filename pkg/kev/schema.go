package kev

import "github.com/xeipuuv/gojsonschema"

func excludeTypes(ts ...string) func(gojsonschema.ResultError) bool {
	return func(re gojsonschema.ResultError) bool {
		for _, t := range ts {
			if t == re.Type() {
				return false
			}
		}
		return true
	}
}

func withType(t string) func(gojsonschema.ResultError) bool {
	return func(re gojsonschema.ResultError) bool {
		return t == re.Type()
	}
}

func findError(result *gojsonschema.Result, predicate func(re gojsonschema.ResultError) bool) gojsonschema.ResultError {
	if result.Valid() {
		return nil
	}

	for _, e := range result.Errors() {
		if predicate(e) {
			return e
		}
	}

	return nil
}
