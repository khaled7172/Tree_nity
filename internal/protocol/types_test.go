package protocol

import (
	"encoding/json"
	"strings"
	"testing"
)

// testvalidateidentiifier verifies regex validation for topic names and client ID
func TestValidateIdentifier(t *testing.T) {
	validCases := []string{
		"client0",
		"user_events",
		"order-setvice",
		"metrics.cpu",
		"a",
		"A123_.-",
		"exact_32_characters_long_name__",
	}

	for _, name := range validCases {
		if !ValidateIdentifier(name) {
			t.Errorf("ValidateIdentifier(%q) = false; want true", name)
		}
	}

	invalidCases := []string{
		"",
		"has space",
		"has/slash",
		"user@domain",
		"topic!",
		"this_name_is_way_too_long_exceeding_32_characters_strictly",
	}

	for _, name := range invalidCases {
		if ValidateIdentifier(name) {
			t.Errorf("ValidateIdentifier(%q) = true; want false", name)
		}
	}
}

func TestClientInfoJSON(t *testing.T) {
	info := ClientInfo{
		Client: "client0",
		Topic:  "user_events",
		Offset: 4,
		Prefix: "user.create",
		IPC:    "/tmp/operator.server.1337.client0",
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	jsonStr := string(data)

	requiredSubstrings := []string{
		`"client":"client0"`,
		`"topic":"user_events"`,
		`"offset":4`,
		`"prefix":"user.create"`,
		`"ipc":"/tmp/operator.server.1337.client0"`,
	}

	for _, sub := range requiredSubstrings {
		if !strings.Contains(jsonStr, sub) {
			t.Errorf("JSON output missing %q; got: %s", sub, jsonStr)
		}
	}
}

func TestRequestResponseJSON(t *testing.T) {
	req := Request{
		Type:      CmdCreate,
		Topic:     "orders",
		ReplyFIFO: "/tmp/reply.fifo",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal Request failed: %v", err)
	}

	var decodedReq Request
	if err := json.Unmarshal(data, &decodedReq); err != nil {
		t.Fatalf("Unmarshal Request failed: %v", err)
	}

	if decodedReq.Type != CmdCreate || decodedReq.Topic != "orders" {
		t.Errorf("Decoded request mismatch: got %+v", decodedReq)
	}

	resp := Response{
		Success:  true,
		ExitCode: ExitSuccess,
		Message:  "topic created",
	}

	respData, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal Response failed: %v", err)
	}

	var decodedResp Response
	if err := json.Unmarshal(respData, &decodedResp); err != nil {
		t.Fatalf("Unmarshal Response failed: %v", err)
	}

	if !decodedResp.Success || decodedResp.ExitCode != 0 {
		t.Errorf("Decoded response mismatch: got %+v", decodedResp)
	}
}
