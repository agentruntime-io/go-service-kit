package logging

import "go.uber.org/zap"

func AttrsToFields(args ...any) []zap.Field {
	if len(args) == 0 {
		return nil
	}

	fields := make([]zap.Field, 0, len(args))
	for i := 0; i < len(args); i++ {
		if field, ok := args[i].(zap.Field); ok {
			fields = append(fields, field)
			continue
		}
		key, ok := args[i].(string)
		if !ok || i+1 >= len(args) {
			continue
		}
		fields = append(fields, Any(key, args[i+1]))
		i++
	}
	return dedupeFields(fields)
}

func dedupeFields(fields []zap.Field) []zap.Field {
	if len(fields) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(fields))
	out := make([]zap.Field, 0, len(fields))
	for _, field := range fields {
		if field.Key == "" {
			out = append(out, field)
			continue
		}
		if _, exists := seen[field.Key]; exists {
			continue
		}
		seen[field.Key] = struct{}{}
		out = append(out, field)
	}
	return out
}
