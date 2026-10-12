package model

func RateOptionLabel(basis string, unit *string) string {
	unitCode := ""
	if unit != nil {
		unitCode = *unit
	}

	switch basis {
	case "negotiable":
		return "Negotiable"
	case "fixed":
		return "Fixed contract"
	case "per_time":
		switch unitCode {
		case "day":
			return "Per person / day"
		case "hour":
			return "Per person / hour"
		}
	case "per_output":
		switch unitCode {
		case "kg":
			return "Per kg"
		case "bag":
			return "Per bag"
		}
	case "per_area":
		switch unitCode {
		case "acre":
			return "Per acre"
		case "sq_ft":
			return "Per sq ft"
		}
	case "per_entity":
		if unitCode == "tree" {
			return "Per tree"
		}
	}

	if unitCode != "" {
		return "Per " + unitCode
	}
	return basis
}

func RateOptionMatches(options []RateOption, basis, unit string) bool {
	for _, option := range options {
		optionUnit := ""
		if option.RateUnit != nil {
			optionUnit = *option.RateUnit
		}
		if option.RateBasis == basis && optionUnit == unit {
			return true
		}
	}
	return false
}

type RateOption struct {
	RateBasis     string  `json:"rate_basis"`
	RateBasisName string  `json:"rate_basis_name"`
	RateUnit      *string `json:"rate_unit,omitempty"`
	RateUnitName  *string `json:"rate_unit_name,omitempty"`
	IsDefault     bool    `json:"is_default"`
	Label         string  `json:"label"`
}
