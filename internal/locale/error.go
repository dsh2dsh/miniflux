// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package locale

type Error interface {
	error

	Translate(string) string
}

type LocalizedErrorWrapper struct {
	err             error
	translationKey  string
	translationArgs []any
}

var _ Error = (*LocalizedErrorWrapper)(nil)

func NewLocalizedErrorWrapper(originalErr error, translationKey string,
	translationArgs ...any,
) *LocalizedErrorWrapper {
	return &LocalizedErrorWrapper{
		err:             originalErr,
		translationKey:  translationKey,
		translationArgs: translationArgs,
	}
}

func (l *LocalizedErrorWrapper) Error() string { return l.err.Error() }
func (l *LocalizedErrorWrapper) Unwrap() error { return l.err }

func (l *LocalizedErrorWrapper) Translate(language string) string {
	if l.translationKey == "" {
		if l.err == nil {
			return ""
		}
		return l.Error()
	}
	return NewPrinter(language).Printf(l.translationKey, l.translationArgs...)
}

type LocalizedError struct {
	translationKey  string
	translationArgs []any
}

var _ Error = (*LocalizedError)(nil)

func NewLocalizedError(translationKey string, translationArgs ...any,
) *LocalizedError {
	return &LocalizedError{
		translationKey:  translationKey,
		translationArgs: translationArgs,
	}
}

func (v *LocalizedError) Error() string { return v.Translate("en_US") }

func (v *LocalizedError) Translate(language string) string {
	return NewPrinter(language).Printf(v.translationKey, v.translationArgs...)
}
