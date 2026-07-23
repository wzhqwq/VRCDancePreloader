package input

import (
	"fmt"
	"strconv"
	"strings"
)

type InputFilter func(oldText, newText string) (accepted bool, fixedText string)

type ValueValidator func(text string) error

type InputPolicy interface {
	Filter(oldText, newText string) (bool, string)
	Validate(text string) error
}

type policy struct {
	f InputFilter
	v ValueValidator
}

func (p policy) Filter(oldText, newText string) (bool, string) {
	return p.f(oldText, newText)
}

func (p policy) Validate(text string) error {
	return p.v(text)
}

func NewPolicy(f InputFilter, v ValueValidator) InputPolicy {
	return policy{f, v}
}

func nopPolicy() InputPolicy {
	return policy{
		f: func(oldText, newText string) (bool, string) {
			return true, newText
		},
		v: func(text string) error {
			return nil
		},
	}
}

func IntegerFilter(allowNegative bool) InputFilter {
	return func(oldText, newText string) (bool, string) {
		if newText == "" {
			return true, newText
		}

		for i, r := range newText {
			if r >= '0' && r <= '9' {
				continue
			}

			if allowNegative && r == '-' && i == 0 {
				continue
			}

			return false, oldText
		}

		if !allowNegative && newText[0] == '-' {
			return false, oldText
		}

		if allowNegative && strings.Count(newText, "-") > 1 {
			return false, oldText
		}

		return true, newText
	}
}

func FloatFilter(allowNegative bool) InputFilter {
	return func(oldText, newText string) (bool, string) {
		if newText == "" {
			return true, newText
		}

		dotCount := 0
		minusCount := 0

		for i, r := range newText {
			switch {
			case r >= '0' && r <= '9':
				continue

			case r == '.':
				dotCount++
				if dotCount > 1 {
					return false, oldText
				}

			case allowNegative && r == '-' && i == 0:
				minusCount++
				if minusCount > 1 {
					return false, oldText
				}

			default:
				return false, oldText
			}
		}

		return true, newText
	}
}

func IntValidator(allowEmpty bool) ValueValidator {
	return func(text string) error {
		if text == "" {
			if allowEmpty {
				return nil
			}
			return fmt.Errorf("empty value")
		}

		if text == "-" {
			return fmt.Errorf("incomplete integer")
		}

		_, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			return err
		}

		return nil
	}
}

func IntRangeValidator(min, max int64, allowEmpty bool) ValueValidator {
	return func(text string) error {
		if text == "" {
			if allowEmpty {
				return nil
			}
			return fmt.Errorf("empty value")
		}

		if text == "-" {
			return fmt.Errorf("incomplete integer")
		}

		v, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			return err
		}

		if v < min || v > max {
			return fmt.Errorf("value must be between %d and %d", min, max)
		}

		return nil
	}
}

func FloatValidator(allowEmpty bool) ValueValidator {
	return func(text string) error {
		if text == "" {
			if allowEmpty {
				return nil
			}
			return fmt.Errorf("empty value")
		}

		switch text {
		case "-", ".", "-.":
			return fmt.Errorf("incomplete float")
		}

		_, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return err
		}

		return nil
	}
}

func FloatRangeValidator(min, max float64, allowEmpty bool) ValueValidator {
	return func(text string) error {
		if text == "" {
			if allowEmpty {
				return nil
			}
			return fmt.Errorf("empty value")
		}

		switch text {
		case "-", ".", "-.":
			return fmt.Errorf("incomplete float")
		}

		v, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return err
		}

		if v < min || v > max {
			return fmt.Errorf("value must be between %g and %g", min, max)
		}

		return nil
	}
}
