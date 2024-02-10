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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test.convertToFieldErrorToErrorList", func() {
	var (
		ctx                context.Context
		err                FieldError
		path               *field.Path
		ignorePathPrefix   *field.Path
		fieldPath          *field.Path
		errs               field.ErrorList
		expectedErrMessage string
	)

	BeforeEach(func() {
		ctx = context.TODO()
		err = FieldError{}
		path = &field.Path{}
		ignorePathPrefix = &field.Path{}
		fieldPath = &field.Path{}
		errs = field.ErrorList{}
		expectedErrMessage = ""
	})

	JustBeforeEach(func() {
		errs = convertToFieldErrorToErrorList(ctx, &err, path, ignorePathPrefix)
	})

	When("err.Message contains special texts", func() {
		BeforeEach(func() {
			err.Message = "expected exactly one, got neither"
			err.Paths = []string{"field1", "field2"}
		})

		When("converting to field error", func() {
			It("should set correct error message", func() {
				expectedErrMessage = "expected exactly one, got neither: field1, field2"
				Expect(errs).To(HaveLen(1))
				Expect(errs[0].Detail).To(Equal(expectedErrMessage))
			})

			It("should clear error paths", func() {
				Expect(errs[0].Field).To(Equal(fieldPath.String()))
			})
		})
	})

	When("err.Message contains specific texts", func() {
		When("err.Message contains 'missing field(s)'", func() {
			BeforeEach(func() {
				err.Message = "missing field(s)"
			})

			It("should return field.Required error", func() {
				Expect(errs).To(HaveLen(1))
				Expect(errs[0]).To(Equal(field.Required(fieldPath, err.Message)))
			})
		})

		When("err.Message contains 'invalid value: '", func() {
			BeforeEach(func() {
				err.Message = "invalid value: abc"
			})

			It("should return field.Invalid error", func() {
				value := strings.TrimPrefix(err.Message, "invalid value: ")
				Expect(errs).To(HaveLen(1))
				Expect(errs[0]).To(Equal(field.Invalid(fieldPath, value, err.Message)))
			})
		})

		When("err.Message contains specific texts", func() {
			Context("when 'expected exactly one, got neither'", func() {
				BeforeEach(func() {
					err.Message = "expected exactly one, got neither"
				})

				It("should return field.Forbidden error", func() {
					Expect(errs).To(HaveLen(1))
					Expect(errs[0]).To(Equal(field.Forbidden(fieldPath, err.Message)))
				})
			})

			Context("when 'expected exactly one, got both'", func() {
				BeforeEach(func() {
					err.Message = "expected exactly one, got both"
				})

				It("should return field.Forbidden error", func() {
					Expect(errs).To(HaveLen(1))
					Expect(errs[0]).To(Equal(field.Forbidden(fieldPath, err.Message)))
				})
			})

			Context("when 'must not update deprecated field(s)'", func() {
				BeforeEach(func() {
					err.Message = "must not update deprecated field(s)"
				})

				It("should return field.Forbidden error", func() {
					Expect(errs).To(HaveLen(1))
					Expect(errs[0]).To(Equal(field.Forbidden(fieldPath, err.Message)))
				})
			})

			Context("when 'must not set the field(s)'", func() {
				BeforeEach(func() {
					err.Message = "must not set the field(s)"
				})

				It("should return field.Forbidden error", func() {
					Expect(errs).To(HaveLen(1))
					Expect(errs[0]).To(Equal(field.Forbidden(fieldPath, err.Message)))
				})
			})

			Context("when 'invalid key name '", func() {
				BeforeEach(func() {
					err.Message = "invalid key name abc"
				})

				It("should return field.Forbidden error", func() {
					Expect(errs).To(HaveLen(1))
					Expect(errs[0]).To(Equal(field.Forbidden(fieldPath, err.Message)))
				})
			})
		})

		Context("when err.Message contains 'Internal Error'", func() {
			BeforeEach(func() {
				err.Message = "Internal Error"
			})

			It("should return field.InternalError error", func() {
				Expect(errs).To(HaveLen(1))
				Expect(errs[0]).To(Equal(field.InternalError(fieldPath, errors.New(err.Message))))
			})
		})

		Context("when err.Message does not match any specific text", func() {
			BeforeEach(func() {
				err.Message = "unknown error"
			})

			It("should return field.Invalid error", func() {
				Expect(errs).To(HaveLen(1))
				Expect(errs[0]).To(Equal(field.Invalid(fieldPath, err.Details, err.Message)))
			})
		})
	})

	When("ignorePathPrefix is not empty", func() {
		BeforeEach(func() {
			err.Paths = []string{"prefix", "field1", "field2"}
			ignorePathPrefix = field.NewPath("path")
		})

		When("pathString does not have ignorePathPrefix as prefix", func() {

			It("should return nil error list", func() {
				Expect(errs).To(BeNil())
			})
		})

		When("pathString has ignorePathPrefix as prefix", func() {
			BeforeEach(func() {
				ignorePathPrefix = field.NewPath("prefix")
			})

			It("should set correct error paths", func() {
				expectedPathString := flatten(err.Paths[1:])
				expectedFieldPath := field.NewPath(expectedPathString)
				Expect(errs).To(HaveLen(1))
				Expect(errs[0].Field).To(Equal(expectedFieldPath.String()))
			})
		})
	})

	When("err.Paths is not empty", func() {
		BeforeEach(func() {
			err.Paths = []string{"field1", "field2"}
		})

		When("fieldPath.String() is emptyFieldPathString", func() {
			BeforeEach(func() {
				path = field.NewPath("")
			})

			It("should set correct field path", func() {
				fieldPath = field.NewPath(err.Paths[0], err.Paths[1:]...)
				Expect(errs).To(HaveLen(1))
				Expect(errs[0].Field).To(Equal(fieldPath.String()))
			})
		})

		When("fieldPath.String() is not emptyFieldPathString", func() {
			BeforeEach(func() {
				path = field.NewPath("parent")
			})

			It("should set correct field path", func() {
				fieldPath = path.Child(err.Paths[0], err.Paths[1:]...)
				Expect(errs).To(HaveLen(1))
				Expect(errs[0].Field).To(Equal(fieldPath.String()))
			})
		})
	})

})

var _ = Describe("Test.ConvertToFieldErrorToErrorListIgnorePathPrefix", func() {
	var (
		ctx                          context.Context
		path, ignorePathPrefix       *field.Path
		fieldError                   *FieldError
		errorList, expectedErrorList field.ErrorList
	)

	BeforeEach(func() {
		ctx = context.TODO()
		path = field.NewPath("path")
		ignorePathPrefix = field.NewPath("")
		fieldError = &FieldError{}
		errorList = field.ErrorList{}
		expectedErrorList = field.ErrorList{}
	})

	JustBeforeEach(func() {
		errorList = ConvertToFieldErrorToErrorListIgnorePathPrefix(ctx, fieldError, path, ignorePathPrefix)
	})

	When("dumplicate error exist", func() {
		BeforeEach(func() {
			ignorePathPrefix = field.NewPath("")
			fieldError = &FieldError{
				errors: []FieldError{
					{
						Message: "expected exactly one, got neither",
						Paths:   []string{"field1", "field2"},
					},
					{
						Message: "expected exactly one, got neither",
						Paths:   []string{"field1", "field2"},
					},
					{
						Message: "invalid value: ",
						Paths:   []string{"path1", "path2"},
					},
				},
			}
			expectedErrorList = field.ErrorList{
				{
					Field:    "path",
					Detail:   "expected exactly one, got neither: field1, field2",
					Type:     "FieldValueForbidden",
					BadValue: "",
				},
				{
					Field:    "path.path1.path2",
					Detail:   "invalid value: ",
					Type:     "FieldValueInvalid",
					BadValue: "",
				},
			}
		})
		It("should return error list with only one error", func() {
			Expect(errorList).To(Equal(expectedErrorList))
		})
	})

	When("ignore path prefix exist and error does not match", func() {
		BeforeEach(func() {
			ignorePathPrefix = field.NewPath("prefix")
			fieldError = &FieldError{
				errors: []FieldError{
					{
						Message: "invalid value: ",
						Paths:   []string{"path1", "path2"},
					},
				},
			}
			expectedErrorList = nil
		})
		It("should return error list with only one error", func() {
			Expect(errorList).To(Equal(expectedErrorList))
		})
	})

	When("ignore path prefix exist and error matched", func() {
		BeforeEach(func() {
			ignorePathPrefix = field.NewPath("path1")
			fieldError = &FieldError{
				errors: []FieldError{
					{
						Message: "invalid value: ",
						Paths:   []string{"path1", "path2"},
					},
				},
			}
			expectedErrorList = field.ErrorList{
				{
					Field:    "path.path2",
					Detail:   "invalid value: ",
					Type:     "FieldValueInvalid",
					BadValue: "",
				},
			}
		})
		It("should return error list with only one error", func() {
			Expect(errorList).To(Equal(expectedErrorList))
		})
	})

})
