package pgexcel

// Export-level options

// WithHeaderStyle sets a custom style for header row
func WithHeaderStyle(style *CellStyle) ExportOption {
	return func(cfg *ExportConfig) error {
		cfg.HeaderStyle = style
		return nil
	}
}

// WithDateFormat sets the date format for date columns
func WithDateFormat(format string) ExportOption {
	return func(cfg *ExportConfig) error {
		cfg.DateFormat = format
		return nil
	}
}

// WithTimeFormat sets the time format for time columns
func WithTimeFormat(format string) ExportOption {
	return func(cfg *ExportConfig) error {
		cfg.TimeFormat = format
		return nil
	}
}

// WithNumberFormat sets the number format for numeric columns
func WithNumberFormat(format string) ExportOption {
	return func(cfg *ExportConfig) error {
		cfg.NumberFormat = format
		return nil
	}
}

// WithAutoFilter enables auto-filter on the header row
func WithAutoFilter() ExportOption {
	return func(cfg *ExportConfig) error {
		cfg.AutoFilter = true
		return nil
	}
}

// WithFreezePanes freezes the header row
func WithFreezePanes() ExportOption {
	return func(cfg *ExportConfig) error {
		cfg.FreezeHeader = true
		return nil
	}
}

// WithAutoFitColumns enables auto-fitting column widths
func WithAutoFitColumns() ExportOption {
	return func(cfg *ExportConfig) error {
		cfg.AutoFitColumns = true
		return nil
	}
}

// WithMaxColumnWidth sets the maximum column width
func WithMaxColumnWidth(width int) ExportOption {
	return func(cfg *ExportConfig) error {
		cfg.MaxColumnWidth = width
		return nil
	}
}

// WithProtection sets the protection configuration for the sheet
func WithProtection(protection *SheetProtection) ExportOption {
	return func(cfg *ExportConfig) error {
		cfg.Protection = protection
		return nil
	}
}

// WithProtectionRules builds protection from rules
func WithProtectionRules(password string, rules ...ProtectionRule) ExportOption {
	return func(cfg *ExportConfig) error {
		sp := NewSheetProtection()
		sp.Password = password

		for _, rule := range rules {
			if err := rule.Apply(sp); err != nil {
				return err
			}
		}

		cfg.Protection = sp
		return nil
	}
}

// WithColumnStyle sets a custom style for a specific column
func WithColumnStyle(columnName string, style *CellStyle) ExportOption {
	return func(cfg *ExportConfig) error {
		if cfg.DataStyles == nil {
			cfg.DataStyles = make(map[string]*CellStyle)
		}
		cfg.DataStyles[columnName] = style
		return nil
	}
}

// WithHeaders enables or disables header row (default: enabled)
func WithHeaders(include bool) ExportOption {
	return func(cfg *ExportConfig) error {
		cfg.IncludeHeaders = include
		return nil
	}
}

// Sheet-level options

// WithSheetProtection sets protection for this specific sheet
func WithSheetProtection(protection *SheetProtection) SheetOption {
	return func(cfg *SheetConfig) error {
		cfg.Protection = protection
		return nil
	}
}

// WithSheetProtectionRules builds sheet protection from rules
func WithSheetProtectionRules(password string, rules ...ProtectionRule) SheetOption {
	return func(cfg *SheetConfig) error {
		sp := NewSheetProtection()
		sp.Password = password

		for _, rule := range rules {
			if err := rule.Apply(sp); err != nil {
				return err
			}
		}

		cfg.Protection = sp
		return nil
	}
}

// WithQueryArgs sets the query arguments for this sheet
func WithQueryArgs(args ...interface{}) SheetOption {
	return func(cfg *SheetConfig) error {
		cfg.Args = args
		return nil
	}
}
