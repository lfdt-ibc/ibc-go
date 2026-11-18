// SPDX-License-Identifier: Apache-2.0

package logging_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/log/v2"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/lfdt-ibc/ibc-go/v11/internal/logging"
)

func TestSdkEventsToLogArguments(t *testing.T) {
	events := sdk.Events{
		sdk.NewEvent("channel_open_init",
			sdk.NewAttribute("port_id", "icacontroller-owner"),
			sdk.NewAttribute("channel_id", "channel-0"),
			sdk.NewAttribute("counterparty_channel_id", ""),
		),
		sdk.NewEvent("message", sdk.NewAttribute("module", "ibc_channel")),
	}

	args := logging.SdkEventsToLogArguments(events)
	require.Equal(t, []any{
		"events",
		[]string{"type=channel_open_init", "port_id=icacontroller-owner", "channel_id=channel-0", "type=message", "module=ibc_channel"},
	}, args)

	var buf bytes.Buffer
	logger := log.NewLogger(&buf, log.OutputJSONOption(), log.LevelOption(-1))
	logger.Debug("emitting events", args...)
	require.Contains(t, buf.String(), `"events":["type=channel_open_init","port_id=icacontroller-owner","channel_id=channel-0","type=message","module=ibc_channel"]`)
}
