package workspaces

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See NOTICE.txt in the project root for license information.

type OutboundRule interface {
}

func unmarshalOutboundRuleImplementation(input []byte) (OutboundRule, error) {
	if input == nil {
		return nil, nil
	}

	var temp map[string]interface{}
	if err := json.Unmarshal(input, &temp); err != nil {
		return nil, fmt.Errorf("unmarshaling OutboundRule into map[string]interface: %+v", err)
	}

	value, ok := temp["type"].(string)
	if !ok {
		return nil, nil
	}

	if strings.EqualFold(value, "FQDN") {
		var out FqdnOutboundRule
		if err := json.Unmarshal(input, &out); err != nil {
			return nil, fmt.Errorf("unmarshaling into FqdnOutboundRule: %+v", err)
		}
		return out, nil
	}

	if strings.EqualFold(value, "PrivateEndpoint") {
		var out PrivateEndpointOutboundRule
		if err := json.Unmarshal(input, &out); err != nil {
			return nil, fmt.Errorf("unmarshaling into PrivateEndpointOutboundRule: %+v", err)
		}
		return out, nil
	}

	if strings.EqualFold(value, "ServiceTag") {
		var out ServiceTagOutboundRule
		if err := json.Unmarshal(input, &out); err != nil {
			return nil, fmt.Errorf("unmarshaling into ServiceTagOutboundRule: %+v", err)
		}
		return out, nil
	}

	type RawOutboundRuleImpl struct {
		Type   string                 `json:"-"`
		Values map[string]interface{} `json:"-"`
	}
	out := RawOutboundRuleImpl{
		Type:   value,
		Values: temp,
	}
	return out, nil

}
