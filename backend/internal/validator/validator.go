// Package validator contains the validator
// for the project
package validator

type Validator struct {
	Errors map[string]string
}

func NewValidator() *Validator {
	return &Validator{
		Errors: map[string]string{},
	}
}

func (v *Validator) Valid() bool {
	return len(v.Errors) == 0
}

func (v *Validator) AddError(key, val string) {
	if _, ok := v.Errors[key]; !ok {
		v.Errors[key] = val
	}
}

func (v *Validator) Assert(ok bool, key, val string) {
	if !ok {
		v.AddError(key, val)
	}
}
