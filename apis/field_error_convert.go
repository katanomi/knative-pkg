/*
Copyright 2017 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package apis

import (
	"context"
	"errors"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ConvertToFieldErrorToErrorList converts a FieldError into a field.ErrorList
func ConvertToFieldErrorToErrorList(ctx context.Context, err *FieldError, path *field.Path) (errs field.ErrorList) {
	if err == nil {
		return
	}
	if len(err.errors) > 0 {
		for i, oneErr := range err.errors {
			if len(oneErr.errors) > 0 {
				errs = append(errs, ConvertToFieldErrorToErrorList(ctx, &err.errors[i], path)...)
			} else {
				errs = append(errs, convertToFieldError(ctx, oneErr, path)...)
			}
		}
		return
	}
	errs = append(errs, convertToFieldError(ctx, *err, path)...)
	return
}

var emptyFieldPathString = field.NewPath("").String()

func convertToFieldError(ctx context.Context, err FieldError, path *field.Path) (errs field.ErrorList) {
	fieldPath := path
	if len(err.Paths) > 0 {
		// skip first empty path
		if fieldPath.String() == emptyFieldPathString {
			fieldPath = field.NewPath(err.Paths[0])
		} else {
			fieldPath = fieldPath.Child(err.Paths[0])
		}
	}
	// checking which error
	var fieldErr *field.Error
	switch {
	case strings.Contains(err.Message, "missing field(s)"):
		fieldErr = field.Required(fieldPath, err.Message)
	case strings.Contains(err.Message, "invalid value: "):
		value := strings.TrimPrefix(err.Message, "invalid value: ")
		fieldErr = field.Invalid(fieldPath, value, err.Message)
	case strings.Contains(err.Message, "expected exactly one, got neither"),
		strings.Contains(err.Message, "expected exactly one, got both"),
		strings.Contains(err.Message, "must not update deprecated field(s)"),
		strings.Contains(err.Message, "must not set the field(s)"),
		strings.Contains(err.Message, "invalid key name "):
		fieldErr = field.Forbidden(fieldPath, err.Message)
	case strings.Contains(err.Message, "Internal Error"):
		fieldErr = field.InternalError(fieldPath, errors.New(err.Message))
	default:
		fieldErr = field.Invalid(fieldPath, err.Details, err.Message)
	}
	if fieldErr != nil {
		errs = append(errs, fieldErr)
	}
	return
}
