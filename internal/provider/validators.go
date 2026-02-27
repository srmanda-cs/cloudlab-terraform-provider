package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// ---------------------------------------------------------------------------
// RFC3339 string validator
// ---------------------------------------------------------------------------

// rfc3339Validator verifies that a string attribute is a valid RFC3339
// timestamp (e.g. "2026-06-01T00:00:00Z").
type rfc3339Validator struct{}

// validateRFC3339 returns a validator.String that rejects non-RFC3339 values.
func validateRFC3339() validator.String {
	return rfc3339Validator{}
}

func (v rfc3339Validator) Description(_ context.Context) string {
	return "value must be a valid RFC3339 timestamp (e.g. 2006-01-02T15:04:05Z)"
}

func (v rfc3339Validator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v rfc3339Validator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := req.ConfigValue.ValueString()
	if _, err := time.Parse(time.RFC3339, val); err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid RFC3339 Timestamp",
			fmt.Sprintf("The value %q is not a valid RFC3339 timestamp: %s", val, err),
		)
	}
}
