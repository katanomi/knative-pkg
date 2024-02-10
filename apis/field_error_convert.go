/*
Copyright 2024 The Knative Authors

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

// ConvertToFieldErrorToErrorListIgnorePathPrefix converts a FieldError into a field.ErrorList
// and ignores the path prefix if the error path not starts with the prefix
// and removes the prefix from the path
func ConvertToFieldErrorToErrorListIgnorePathPrefix(ctx context.Context, err *FieldError, path, ignorePathPrefix *field.Path) (errs field.ErrorList) {
	errs = convertToFieldErrorToErrorList(ctx, err, path, ignorePathPrefix)
	// Upgrading tekton to v0.56, errors may repeat
	// In tekton, it merged list of FieldErrors, remove duplicate errors
	// Ref: https://github.com/knative/pkg/blob/f5b42e8dea446a2a695ded0ea7c445317aed78b3/apis/field_error.go#L341-L345
	errs = removeDumplicateError(errs)
	return
}

// ConvertToFieldErrorToErrorList converts a FieldError into a field.ErrorList
func ConvertToFieldErrorToErrorList(ctx context.Context, err *FieldError, path *field.Path) (errs field.ErrorList) {
	return ConvertToFieldErrorToErrorListIgnorePathPrefix(ctx, err, path, nil)
}

// convertToFieldErrorToErrorList converts a FieldError into a field.ErrorList
func convertToFieldErrorToErrorList(ctx context.Context, err *FieldError, path, ignorePathPrefix *field.Path) (errs field.ErrorList) {
	if err == nil {
		return
	}
	if len(err.errors) > 0 {
		for i, oneErr := range err.errors {
			if len(oneErr.errors) > 0 {
				errs = append(errs, convertToFieldErrorToErrorList(ctx, &err.errors[i], path, ignorePathPrefix)...)
			} else {
				errs = append(errs, convertToFieldError(ctx, oneErr, path, ignorePathPrefix)...)
			}
		}
		return
	}
	errs = append(errs, convertToFieldError(ctx, *err, path, ignorePathPrefix)...)
	return
}

// removeDumplicateError removes duplicate errors from the list
func removeDumplicateError(errs field.ErrorList) (newErrs field.ErrorList) {
	seen := make(map[string]bool)
	for _, err := range errs {
		errString := err.Error()
		if _, ok := seen[errString]; !ok {
			seen[errString] = true
			newErrs = append(newErrs, err)
		}
	}
	return
}

var emptyFieldPathString = field.NewPath("").String()

func convertToFieldError(_ context.Context, err FieldError, path, ignorePathPrefix *field.Path) (errs field.ErrorList) {
	fieldPath := path

	// this error is a bit special, the paths not really the path
	if strings.Contains(err.Message, "expected exactly one, got neither") ||
		strings.Contains(err.Message, "expected exactly one, got both") ||
		strings.Contains(err.Message, "must not update deprecated field(s)") ||
		strings.Contains(err.Message, "must not set the field(s)") {
		if len(err.Paths) > 0 {
			err.Message += ": " + strings.Join(err.Paths, ", ")
		}
		err.Paths = nil
	}

	pathString := flatten(err.Paths)
	if ignorePathPrefix != nil && ignorePathPrefix.String() != emptyFieldPathString {
		prefix := ignorePathPrefix.String()
		if !strings.HasPrefix(pathString, prefix) {
			return nil
		}
		pathString = strings.TrimPrefix(pathString, prefix)
		if strings.HasPrefix(pathString, ".") {
			pathString = strings.TrimPrefix(pathString, ".")
		}
		err.Paths = []string{pathString}
	}

	if len(err.Paths) > 0 {
		// skip first empty path
		if fieldPath.String() == emptyFieldPathString {
			fieldPath = field.NewPath(err.Paths[0], err.Paths[1:]...)
		} else {
			fieldPath = fieldPath.Child(err.Paths[0], err.Paths[1:]...)
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
