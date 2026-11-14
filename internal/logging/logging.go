// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SdkEventsToLogArguments converts a given sdk.Events into logger key/value arguments. The events are
// logged under the "events" key as a slice of strings that provide human readable values for the event
// attributes.
func SdkEventsToLogArguments(events sdk.Events) []any {
	var attrs []string
	for _, e := range events {
		attrs = append(attrs, fmt.Sprintf("type=%s", e.Type))
		for _, attr := range e.Attributes {
			if len(attr.Value) == 0 {
				continue
			}
			attrs = append(attrs, fmt.Sprintf("%s=%s", attr.Key, attr.Value))
		}
	}
	return []any{"events", attrs}
}
