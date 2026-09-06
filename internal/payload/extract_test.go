package payload

import (
	"encoding/json"
	"testing"
)

func TestExtractJSONFromLogs(t *testing.T) {
	for _, tt := range []struct{ input, want string }{
		{`im_service.send_message send message to im: 900000000000000001, {"AppId":1,"ConversationShortId":900000000000000002,"Content":"{\"text\":\"hello\"}"}`, `{"AppId":1,"Content":"{\"text\":\"hello\"}","ConversationShortId":900000000000000002}`},
		{`[INFO] sent: {"Content":"{\"text\":\"a } [ b\"}"} elapsed=10ms`, `{"Content":"{\"text\":\"a } [ b\"}"}`},
		{`[INFO] sent: {'Content': '{"text":"hello"}', 'ok': True} elapsed=10ms`, `{"Content":"{\"text\":\"hello\"}","ok":true}`},
		{`prefix [1, {"id": 2}] suffix`, `[1,{"id":2}]`},
		{`prefix {items: [1, 2`, `{"items":[1,2]}`},
		{`im_service.send_message send message to im: 900000000000000001, {"Content":"{\\"text\\":\\"hello\\"}","Ext":{"mc:ext_json":"{\\"message_type\\":1,\\"message_commands\\":[{\\"cmd\\":\\"test\\"}]}"}}`, `{"Content":"{\"text\":\"hello\"}","Ext":{"mc:ext_json":"{\"message_type\":1,\"message_commands\":[{\"cmd\":\"test\"}]}"}}`},
		{`"text with {brackets}"`, `"text with {brackets}"`},
	} {
		got, err := FormatJSON(tt.input, true)
		if err != nil || got != tt.want {
			t.Errorf("input %q: got %s (%v), want %s", tt.input, got, err, tt.want)
		}
	}
}

func TestMessageLogKeepsEmbeddedJSONStrings(t *testing.T) {
	// Same nested message shape as the reported log; fixture identifiers are synthetic.
	ext := `{"message_type":1,"message_commands":[{"cmd":"pay_success","payload":{"out_trade_no":"test-order","item_info":[{"goods":[{"item_name":"\u6676\u94bb","item_img":"https://example.invalid/1.png","number":60}]}],"pay_info":{"amount":"6.00","currency":"CNY"},"event_timestamp":1788000000000}}],"streaming_is_finish":true}`
	content := `{"text":"\u7cfb\u7edf\u6d88\u606f"}`
	document := map[string]any{
		"AppId": json.Number("123456"), "ConversationShortId": json.Number("900000000000000002"),
		"Content": content, "Ext": map[string]any{"mc:ext_json": ext},
	}
	encoded, err := MarshalJSON(document, true)
	if err != nil {
		t.Fatal(err)
	}
	value, err := ParseJSON("im_service.send_message send message to im: 900000000000000001, " + encoded)
	if err != nil {
		t.Fatal(err)
	}
	obj := value.(map[string]any)
	if obj["Content"] != content || obj["Ext"].(map[string]any)["mc:ext_json"] != ext {
		t.Fatal("embedded strings changed before manual expansion")
	}
	if obj["ConversationShortId"] != json.Number("900000000000000002") {
		t.Fatal("integer precision lost")
	}
	if expanded, ok := ParseStringJSON(ext); !ok || expanded.(map[string]any)["message_type"] != json.Number("1") {
		t.Fatalf("nested JSON not expandable: %v", expanded)
	}
}

func TestMessageLogWithMarkdownEscapes(t *testing.T) {
	input := `im\_service.send\_message sent: {"Content":"{\\"text\\":\\"system message\\"}","Ext":{"mc:ext\_json":"{\\"message\_type\\":1,\\"message\_commands\\":[{\\"cmd\\":\\"pay\_success\\",\\"payload\\":{\\"url\\":\\"[https://example.invalid/1.png\\](https://example.invalid/1.png\\)\\"}}]}"}}`
	value, err := ParseJSON(input)
	if err != nil {
		t.Fatal(err)
	}
	obj := value.(map[string]any)
	content, ok := obj["Content"].(string)
	if !ok {
		t.Fatal("Content no longer a string")
	}
	if parsed, ok := ParseStringJSON(content); !ok || parsed.(map[string]any)["text"] != "system message" {
		t.Fatalf("cannot expand Content: %q", content)
	}
	ext := obj["Ext"].(map[string]any)["mc:ext_json"].(string)
	parsed, ok := ParseStringJSON(ext)
	if !ok || parsed.(map[string]any)["message_type"] != json.Number("1") {
		t.Fatalf("cannot expand Ext: %q", ext)
	}
	commands := parsed.(map[string]any)["message_commands"].([]any)
	if commands[0].(map[string]any)["cmd"] != "pay_success" {
		t.Fatal("nested command changed")
	}
}

func TestExpandMessageWithUnmatchedClosingBracket(t *testing.T) {
	inner := `{"message_type":1,"message_commands":{"cmd":"pay_success","payload":{"item_info":[{"goods":[{"item_img":"[https://example.invalid/1.png","number":60}]}],"event_timestamp":1788000000000}}],"chat_title":"","message_merge_finished":true,"show_status":2}`
	document, _ := MarshalJSON(map[string]any{"Content": `{"text":"system message"}`, "Ext": map[string]any{"mc:ext_json": inner}}, true)
	value, err := ParseJSON("```swift\nim_service.send_message sent: " + document + "\nsupported\n```")
	if err != nil {
		t.Fatal(err)
	}
	stored := value.(map[string]any)["Ext"].(map[string]any)["mc:ext_json"]
	if stored != inner {
		t.Fatal("inner string changed before expansion")
	}
	expanded, err := parseDocument(stored.(string))
	if err != nil {
		t.Fatal(err)
	}
	obj := expanded.(map[string]any)
	if obj["show_status"] != json.Number("2") || obj["message_merge_finished"] != true {
		t.Fatal("repair dropped fields after the unmatched bracket")
	}
	command := obj["message_commands"].(map[string]any)
	goods := command["payload"].(map[string]any)["item_info"].([]any)[0].(map[string]any)["goods"].([]any)
	if goods[0].(map[string]any)["item_img"] != "[https://example.invalid/1.png" {
		t.Fatal("repair changed a bracket inside a string")
	}
}
