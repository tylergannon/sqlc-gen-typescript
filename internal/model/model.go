package model

import "strings"

type TypeRef struct {
	Name     string
	Nullable bool
	ArrayDim int
}

func (t TypeRef) Render() string {
	var base strings.Builder
	base.WriteString(t.Name)
	for range t.ArrayDim {
		base.WriteString("[]")
	}
	if t.Nullable {
		return base.String() + " | null"
	}
	return base.String()
}

func FieldName(prefix string, index int, name string) string {
	if name == "" {
		name = prefix + "_" + strconvItoa(index)
	}
	lower := strings.ToLower(name)
	var b strings.Builder
	for i := 0; i < len(lower); i++ {
		if lower[i] == '_' && i+1 < len(lower) && lower[i+1] >= 'a' && lower[i+1] <= 'z' {
			b.WriteByte(lower[i+1] - 'a' + 'A')
			i++
			continue
		}
		b.WriteByte(lower[i])
	}
	return b.String()
}

func ArgName(index int, name string) string {
	return FieldName("arg", index, name)
}

func ColName(index int, name string) string {
	return FieldName("col", index, name)
}

func LowerTitle(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func PascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	return strings.Join(parts, "")
}

func strconvItoa(i int) string {
	if i == 0 {
		return "0"
	}
	var digits [20]byte
	pos := len(digits)
	n := i
	for n > 0 {
		pos--
		digits[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(digits[pos:])
}
