package engine

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"Tree_nity/internal/hashmap"
	"Tree_nity/internal/ipc"
	"Tree_nity/internal/protocol"
	"Tree_nity/internal/topic"
)

func sendRequest(t *testing.T, serverFIFO string, req protocol.Request) protocol.Response {
	tmpDir := t.TempDir()
	replyFIFO := filepath.Join(tmpDir, "reply.fifo")
	if err := ipc.MakeFifo(replyFIFO); err != nil {
		t.Fatalf("MakeFifo failed: %v", err)
	}
	defer ipc.RemoveFifo(replyFIFO)

	req.ReplyFIFO = replyFIFO
	reqBytes, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	reqBytes = append(reqBytes, '\n')

	w, err := ipc.OpenWriter(serverFIFO)
	if err != nil {
		t.Fatalf("open server fifo failed: %v", err)
	}
	if _, err := w.Write(reqBytes); err != nil {
		_ = w.Close()
		t.Fatalf("write to server fifo failed: %v", err)
	}
	_ = w.Close()

	r, err := ipc.OpenReader(replyFIFO)
	if err != nil {
		t.Fatalf("open reply fifo failed: %v", err)
	}
	defer r.Close()

	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		t.Fatalf("no response received on reply fifo")
	}

	var resp protocol.Response
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	return resp
}

func TestServerCreateAndList(t *testing.T) {
	store := topic.New()
	clientMap := hashmap.New()
	srv := NewServer(os.Getpid()+1000, store, clientMap)

	if err := srv.Start(); err != nil {
		t.Fatalf("srv.Start failed: %v", err)
	}
	defer srv.Stop()

	resp := sendRequest(t, srv.Endpoint, protocol.Request{
		Type:  protocol.CmdCreate,
		Topic: "user_events",
	})
	if !resp.Success || resp.ExitCode != protocol.ExitSuccess {
		t.Fatalf("create topic failed: %+v", resp)
	}

	respDup := sendRequest(t, srv.Endpoint, protocol.Request{
		Type:  protocol.CmdCreate,
		Topic: "user_events",
	})
	if respDup.Success || respDup.ExitCode != protocol.ExitTopic {
		t.Fatalf("expected duplicate topic error (code 2), got: %+v", respDup)
	}

	respList := sendRequest(t, srv.Endpoint, protocol.Request{
		Type: protocol.CmdList,
	})
	if !respList.Success || len(respList.Topics) != 1 || respList.Topics[0] != "user_events" {
		t.Fatalf("list topics failed: %+v", respList)
	}
}

func TestServerSubscribeAndInfo(t *testing.T) {
	store := topic.New()
	clientMap := hashmap.New()
	srv := NewServer(os.Getpid()+2000, store, clientMap)

	if err := srv.Start(); err != nil {
		t.Fatalf("srv.Start failed: %v", err)
	}
	defer srv.Stop()

	_ = sendRequest(t, srv.Endpoint, protocol.Request{
		Type:  protocol.CmdCreate,
		Topic: "telemetry",
	})

	respSub := sendRequest(t, srv.Endpoint, protocol.Request{
		Type:     protocol.CmdSubscribe,
		Topic:    "telemetry",
		ClientID: "client0",
		Prefix:   "sensor.",
	})
	if !respSub.Success || respSub.ExitCode != protocol.ExitSuccess {
		t.Fatalf("subscribe failed: %+v", respSub)
	}

	respInfo := sendRequest(t, srv.Endpoint, protocol.Request{
		Type:     protocol.CmdInfo,
		ClientID: "client0",
	})
	if !respInfo.Success || respInfo.Metadata == nil || respInfo.Metadata.Client != "client0" {
		t.Fatalf("info failed: %+v", respInfo)
	}

	respDup := sendRequest(t, srv.Endpoint, protocol.Request{
		Type:     protocol.CmdSubscribe,
		Topic:    "telemetry",
		ClientID: "client0",
	})
	if respDup.Success || respDup.ExitCode != protocol.ExitTopic {
		t.Fatalf("expected duplicate subscriber error (code 2), got: %+v", respDup)
	}
}
