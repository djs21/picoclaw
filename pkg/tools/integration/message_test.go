package integrationtools

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/session"
)

func TestMessageTool_Execute_Success(t *testing.T) {
	tool := NewMessageTool()

	var sentChannel, sentChatID, sentContent string
	tool.SetSendCallback(func(
		ctx context.Context,
		channel, chatID, content, replyToMessageID string,
		mediaParts []bus.MediaPart,
	) error {
		sentChannel = channel
		sentChatID = chatID
		sentContent = content
		if len(mediaParts) != 0 {
			t.Fatalf("expected no media parts, got %d", len(mediaParts))
		}
		if ToolAgentID(ctx) != "" || ToolSessionKey(ctx) != "" || ToolSessionScope(ctx) != nil {
			t.Fatalf("expected empty turn metadata in basic context, got agent=%q session=%q scope=%+v",
				ToolAgentID(ctx), ToolSessionKey(ctx), ToolSessionScope(ctx))
		}
		return nil
	})

	ctx := WithToolContext(context.Background(), "test-channel", "test-chat-id")
	args := map[string]any{
		"content": "Hello, world!",
	}

	result := tool.Execute(ctx, args)

	// Verify message was sent with correct parameters
	if sentChannel != "test-channel" {
		t.Errorf("Expected channel 'test-channel', got '%s'", sentChannel)
	}
	if sentChatID != "test-chat-id" {
		t.Errorf("Expected chatID 'test-chat-id', got '%s'", sentChatID)
	}
	if sentContent != "Hello, world!" {
		t.Errorf("Expected content 'Hello, world!', got '%s'", sentContent)
	}

	// Verify ToolResult meets US-011 criteria:
	// - Send success returns SilentResult (Silent=true)
	if !result.Silent {
		t.Error("Expected Silent=true for successful send")
	}

	// - ForLLM contains send status description
	if result.ForLLM != "Message sent to test-channel:test-chat-id" {
		t.Errorf("Expected ForLLM 'Message sent to test-channel:test-chat-id', got '%s'", result.ForLLM)
	}

	// - ForUser is empty (user already received message directly)
	if result.ForUser != "" {
		t.Errorf("Expected ForUser to be empty, got '%s'", result.ForUser)
	}

	// - IsError should be false
	if result.IsError {
		t.Error("Expected IsError=false for successful send")
	}
}

func TestMessageTool_Execute_WithCustomChannel(t *testing.T) {
	tool := NewMessageTool()

	var sentChannel, sentChatID string
	tool.SetSendCallback(func(
		ctx context.Context,
		channel, chatID, content, replyToMessageID string,
		mediaParts []bus.MediaPart,
	) error {
		sentChannel = channel
		sentChatID = chatID
		return nil
	})

	ctx := WithToolContext(context.Background(), "default-channel", "default-chat-id")
	args := map[string]any{
		"content": "Test message",
		"channel": "custom-channel",
		"chat_id": "custom-chat-id",
	}

	result := tool.Execute(ctx, args)

	// Verify custom channel/chatID were used instead of defaults
	if sentChannel != "custom-channel" {
		t.Errorf("Expected channel 'custom-channel', got '%s'", sentChannel)
	}
	if sentChatID != "custom-chat-id" {
		t.Errorf("Expected chatID 'custom-chat-id', got '%s'", sentChatID)
	}

	if !result.Silent {
		t.Error("Expected Silent=true")
	}
	if result.ForLLM != "Message sent to custom-channel:custom-chat-id" {
		t.Errorf("Expected ForLLM 'Message sent to custom-channel:custom-chat-id', got '%s'", result.ForLLM)
	}
}

func TestMessageTool_Execute_SendFailure(t *testing.T) {
	tool := NewMessageTool()

	sendErr := errors.New("network error")
	tool.SetSendCallback(func(
		ctx context.Context,
		channel, chatID, content, replyToMessageID string,
		mediaParts []bus.MediaPart,
	) error {
		return sendErr
	})

	ctx := WithToolContext(context.Background(), "test-channel", "test-chat-id")
	args := map[string]any{
		"content": "Test message",
	}

	result := tool.Execute(ctx, args)

	// Verify ToolResult for send failure:
	// - Send failure returns ErrorResult (IsError=true)
	if !result.IsError {
		t.Error("Expected IsError=true for failed send")
	}

	// - ForLLM contains error description
	expectedErrMsg := "sending message: network error"
	if result.ForLLM != expectedErrMsg {
		t.Errorf("Expected ForLLM '%s', got '%s'", expectedErrMsg, result.ForLLM)
	}

	// - Err field should contain original error
	if result.Err == nil {
		t.Error("Expected Err to be set")
	}
	if result.Err != sendErr {
		t.Errorf("Expected Err to be sendErr, got %v", result.Err)
	}
}

func TestMessageTool_Execute_MissingContent(t *testing.T) {
	tool := NewMessageTool()

	ctx := WithToolContext(context.Background(), "test-channel", "test-chat-id")
	args := map[string]any{} // content missing

	result := tool.Execute(ctx, args)

	// Verify error result for missing content/media/file_path
	if !result.IsError {
		t.Error("Expected IsError=true for missing content/media/file_path")
	}
	if result.ForLLM != "content, file_path, or media is required" {
		t.Errorf("Expected ForLLM 'content, file_path, or media is required', got '%s'", result.ForLLM)
	}
}

func TestMessageTool_Execute_NoTargetChannel(t *testing.T) {
	tool := NewMessageTool()
	// No WithToolContext — channel/chatID are empty

	tool.SetSendCallback(func(
		ctx context.Context,
		channel, chatID, content, replyToMessageID string,
		mediaParts []bus.MediaPart,
	) error {
		return nil
	})

	ctx := context.Background()
	args := map[string]any{
		"content": "Test message",
	}

	result := tool.Execute(ctx, args)

	// Verify error when no target channel specified
	if !result.IsError {
		t.Error("Expected IsError=true when no target channel")
	}
	if result.ForLLM != "No target channel/chat specified" {
		t.Errorf("Expected ForLLM 'No target channel/chat specified', got '%s'", result.ForLLM)
	}
}

func TestMessageTool_Execute_NotConfigured(t *testing.T) {
	tool := NewMessageTool()
	// No SetSendCallback called

	ctx := WithToolContext(context.Background(), "test-channel", "test-chat-id")
	args := map[string]any{
		"content": "Test message",
	}

	result := tool.Execute(ctx, args)

	// Verify error when send callback not configured
	if !result.IsError {
		t.Error("Expected IsError=true when send callback not configured")
	}
	if result.ForLLM != "Message sending not configured" {
		t.Errorf("Expected ForLLM 'Message sending not configured', got '%s'", result.ForLLM)
	}
}

func TestMessageTool_Name(t *testing.T) {
	tool := NewMessageTool()
	if tool.Name() != "message" {
		t.Errorf("Expected name 'message', got '%s'", tool.Name())
	}
}

func TestMessageTool_Description(t *testing.T) {
	tool := NewMessageTool()
	desc := tool.Description()
	if desc == "" {
		t.Error("Description should not be empty")
	}
}

func TestMessageTool_Parameters(t *testing.T) {
	tool := NewMessageTool()
	params := tool.Parameters()

	// Verify parameters structure
	typ, ok := params["type"].(string)
	if !ok || typ != "object" {
		t.Error("Expected type 'object'")
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("Expected properties to be a map")
	}

	// Check required properties
	required, ok := params["required"].([]string)
	if !ok || len(required) != 1 || required[0] != "content" {
		t.Fatal("Expected content-only required schema when local media is disabled")
	}

	// Check content property
	contentProp, ok := props["content"].(map[string]any)
	if !ok {
		t.Error("Expected 'content' property")
	}
	if contentProp["type"] != "string" {
		t.Error("Expected content type to be 'string'")
	}

	if _, hasMedia := props["media"]; hasMedia {
		t.Fatal("did not expect 'media' property when local media is disabled")
	}

	// Check channel property (optional)
	channelProp, ok := props["channel"].(map[string]any)
	if !ok {
		t.Error("Expected 'channel' property")
	}
	if channelProp["type"] != "string" {
		t.Error("Expected channel type to be 'string'")
	}

	// Check chat_id property (optional)
	chatIDProp, ok := props["chat_id"].(map[string]any)
	if !ok {
		t.Error("Expected 'chat_id' property")
	}
	if chatIDProp["type"] != "string" {
		t.Error("Expected chat_id type to be 'string'")
	}

	// Check reply_to_message_id property (optional)
	replyToProp, ok := props["reply_to_message_id"].(map[string]any)
	if !ok {
		t.Error("Expected 'reply_to_message_id' property")
	}
	if replyToProp["type"] != "string" {
		t.Error("Expected reply_to_message_id type to be 'string'")
	}
}

func TestMessageTool_Parameters_WithLocalMediaEnabled(t *testing.T) {
	tool := NewMessageTool()
	tool.ConfigureLocalMedia(t.TempDir(), true, 1024*1024, nil)
	params := tool.Parameters()

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("Expected properties to be a map")
	}
	mediaProp, ok := props["media"].(map[string]any)
	if !ok {
		t.Fatal("Expected 'media' property")
	}
	if mediaProp["type"] != "array" {
		t.Error("Expected media type to be 'array'")
	}
	anyOf, ok := params["anyOf"].([]map[string]any)
	if !ok || len(anyOf) != 3 {
		t.Fatal("Expected anyOf content/media/file_path requirement")
	}
	if _, ok := params["required"]; ok {
		t.Fatal("did not expect top-level required content when media is enabled")
	}
}

func TestMessageTool_Execute_WithMediaDisabled(t *testing.T) {
	tool := NewMessageTool()
	tool.SetSendCallback(func(
		ctx context.Context,
		channel, chatID, content, replyToMessageID string,
		mediaParts []bus.MediaPart,
	) error {
		t.Fatal("send callback should not run when message media is disabled")
		return nil
	})

	ctx := WithToolContext(context.Background(), "telegram", "-1001")
	result := tool.Execute(ctx, map[string]any{
		"media": []any{
			map[string]any{"path": "photo.jpg"},
		},
	})
	if !result.IsError {
		t.Fatal("expected error when message media is disabled")
	}
	if result.ForLLM != "message media attachments are disabled; enable tools.message.media_enabled to send local media through message" {
		t.Fatalf("unexpected error: %q", result.ForLLM)
	}
}

func TestMessageTool_Execute_WithReplyToMessageID(t *testing.T) {
	tool := NewMessageTool()

	var sentReplyTo string
	tool.SetSendCallback(func(
		ctx context.Context,
		channel, chatID, content, replyToMessageID string,
		mediaParts []bus.MediaPart,
	) error {
		sentReplyTo = replyToMessageID
		return nil
	})

	ctx := WithToolContext(context.Background(), "test-channel", "test-chat-id")
	args := map[string]any{
		"content":             "Reply test",
		"reply_to_message_id": "msg-123",
	}

	result := tool.Execute(ctx, args)
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if sentReplyTo != "msg-123" {
		t.Fatalf("expected reply_to_message_id msg-123, got %q", sentReplyTo)
	}
}

func TestMessageTool_Execute_PropagatesTurnSessionMetadata(t *testing.T) {
	tool := NewMessageTool()

	var gotAgentID, gotSessionKey string
	var gotScope *session.SessionScope
	tool.SetSendCallback(func(
		ctx context.Context,
		channel, chatID, content, replyToMessageID string,
		mediaParts []bus.MediaPart,
	) error {
		gotAgentID = ToolAgentID(ctx)
		gotSessionKey = ToolSessionKey(ctx)
		gotScope = ToolSessionScope(ctx)
		return nil
	})

	ctx := WithToolContext(context.Background(), "test-channel", "test-chat-id")
	ctx = WithToolSessionContext(ctx, "main", "sk_v1_tool", &session.SessionScope{
		Version:    session.ScopeVersionV1,
		AgentID:    "main",
		Channel:    "telegram",
		Dimensions: []string{"chat"},
		Values: map[string]string{
			"chat": "direct:test-chat-id",
		},
	})

	result := tool.Execute(ctx, map[string]any{"content": "Hello, world!"})
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if gotAgentID != "main" {
		t.Fatalf("ToolAgentID() = %q, want main", gotAgentID)
	}
	if gotSessionKey != "sk_v1_tool" {
		t.Fatalf("ToolSessionKey() = %q, want sk_v1_tool", gotSessionKey)
	}
	if gotScope == nil || gotScope.Values["chat"] != "direct:test-chat-id" {
		t.Fatalf("ToolSessionScope() = %+v, want chat scope", gotScope)
	}
}

func TestMessageTool_Execute_WithMedia(t *testing.T) {
	tool := NewMessageTool()
	store := media.NewFileMediaStore()
	dir := t.TempDir()
	imgPath := filepath.Join(dir, "photo.jpg")
	if err := os.WriteFile(imgPath, []byte("fake image bytes"), 0o644); err != nil {
		t.Fatalf("write image: %v", err)
	}
	tool.ConfigureLocalMedia(dir, true, 1024*1024, []*regexp.Regexp{})
	tool.SetMediaStore(store)

	var gotContent string
	var gotParts []bus.MediaPart
	tool.SetSendCallback(func(
		ctx context.Context,
		channel, chatID, content, replyToMessageID string,
		mediaParts []bus.MediaPart,
	) error {
		gotContent = content
		gotParts = append([]bus.MediaPart(nil), mediaParts...)
		return nil
	})

	ctx := WithToolContext(context.Background(), "telegram", "-1001")
	result := tool.Execute(ctx, map[string]any{
		"content": "Caption text",
		"media": []any{
			map[string]any{
				"path": imgPath,
			},
		},
	})
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if gotContent != "Caption text" {
		t.Fatalf("content = %q, want Caption text", gotContent)
	}
	if len(gotParts) != 1 {
		t.Fatalf("expected 1 media part, got %d", len(gotParts))
	}
	if gotParts[0].Caption != "Caption text" {
		t.Fatalf("first part caption = %q, want Caption text", gotParts[0].Caption)
	}
	if gotParts[0].Ref == "" {
		t.Fatal("expected media ref to be populated")
	}
	if gotParts[0].Type == "" {
		t.Fatal("expected media type to be inferred")
	}
}

func TestMessageTool_Execute_WithDirectFilePath(t *testing.T) {
	tool := NewMessageTool()
	store := media.NewFileMediaStore()
	dir := t.TempDir()
	docPath := filepath.Join(dir, "document.pdf")
	if err := os.WriteFile(docPath, []byte("%PDF-1.4 sample pdf content"), 0o644); err != nil {
		t.Fatalf("write doc: %v", err)
	}
	tool.ConfigureLocalMedia(dir, true, 1024*1024, []*regexp.Regexp{})
	tool.SetMediaStore(store)

	var gotContent string
	var gotParts []bus.MediaPart
	tool.SetSendCallback(func(
		ctx context.Context,
		channel, chatID, content, replyToMessageID string,
		mediaParts []bus.MediaPart,
	) error {
		gotContent = content
		gotParts = append([]bus.MediaPart(nil), mediaParts...)
		return nil
	})

	ctx := WithToolContext(context.Background(), "telegram", "-1001")
	result := tool.Execute(ctx, map[string]any{
		"file_path": "document.pdf",
		"content":   "Here is your document",
	})
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if gotContent != "Here is your document" {
		t.Fatalf("content = %q, want 'Here is your document'", gotContent)
	}
	if len(gotParts) != 1 {
		t.Fatalf("expected 1 media part from file_path, got %d", len(gotParts))
	}
	if gotParts[0].Filename != "document.pdf" {
		t.Fatalf("filename = %q, want document.pdf", gotParts[0].Filename)
	}
	if gotParts[0].Ref == "" {
		t.Fatal("expected non-empty media ref")
	}
}

func TestMessageTool_Execute_FilePathWorkspaceRestriction(t *testing.T) {
	tool := NewMessageTool()
	store := media.NewFileMediaStore()
	dir := t.TempDir()
	outsideDir := t.TempDir()
	outsideFile := filepath.Join(outsideDir, "secret.txt")
	if err := os.WriteFile(outsideFile, []byte("secret"), 0o644); err != nil {
		t.Fatalf("write secret: %v", err)
	}

	tool.ConfigureLocalMedia(dir, true, 1024*1024, []*regexp.Regexp{})
	tool.SetMediaStore(store)
	tool.SetSendCallback(func(
		ctx context.Context,
		channel, chatID, content, replyToMessageID string,
		mediaParts []bus.MediaPart,
	) error {
		return nil
	})

	ctx := WithToolContext(context.Background(), "telegram", "-1001")
	result := tool.Execute(ctx, map[string]any{
		"file_path": outsideFile,
	})
	if !result.IsError {
		t.Fatal("expected error due to workspace restriction, got success")
	}
}

func TestMessageTool_Execute_SVGTypedAsFile(t *testing.T) {
	tool := NewMessageTool()
	store := media.NewFileMediaStore()
	dir := t.TempDir()
	svgPath := filepath.Join(dir, "diagram.svg")
	if err := os.WriteFile(svgPath, []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`), 0o644); err != nil {
		t.Fatalf("write svg: %v", err)
	}
	tool.ConfigureLocalMedia(dir, true, 1024*1024, []*regexp.Regexp{})
	tool.SetMediaStore(store)

	var gotParts []bus.MediaPart
	tool.SetSendCallback(func(
		ctx context.Context,
		channel, chatID, content, replyToMessageID string,
		mediaParts []bus.MediaPart,
	) error {
		gotParts = append([]bus.MediaPart(nil), mediaParts...)
		return nil
	})

	ctx := WithToolContext(context.Background(), "telegram", "-1001")
	result := tool.Execute(ctx, map[string]any{"file_path": "diagram.svg"})
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if len(gotParts) != 1 {
		t.Fatalf("expected 1 media part, got %d", len(gotParts))
	}
	// Telegram SendPhoto rejects vector SVG, so it must be delivered as a document.
	if gotParts[0].Type != "file" {
		t.Fatalf("svg part type = %q, want file", gotParts[0].Type)
	}
}

func TestNormalizeMessageMediaType_SVG(t *testing.T) {
	tests := []struct {
		name        string
		typeHint    string
		filename    string
		contentType string
		want        string
	}{
		{"svg content type", "", "diagram.svg", "image/svg+xml", "file"},
		{"svg extension with unknown content type", "", "diagram.svg", "", "file"},
		{"svg extension with octet-stream", "", "diagram.svg", "application/octet-stream", "file"},
		{"svg content type without extension", "", "diagram", "image/svg+xml", "file"},
		{"raster png stays image", "", "photo.png", "image/png", "image"},
		{"explicit hint wins over detection", "image", "photo.png", "image/png", "image"},
		{"explicit file hint on svg", "file", "diagram.svg", "image/svg+xml", "file"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeMessageMediaType(tt.typeHint, tt.filename, tt.contentType)
			if got != tt.want {
				t.Fatalf("normalizeMessageMediaType(%q, %q, %q) = %q, want %q",
					tt.typeHint, tt.filename, tt.contentType, got, tt.want)
			}
		})
	}
}

func TestMessageTool_Execute_DirectoryPath(t *testing.T) {
	tool := NewMessageTool()
	dir := t.TempDir()
	tool.ConfigureLocalMedia(dir, true, 1024*1024, []*regexp.Regexp{})
	tool.SetMediaStore(media.NewFileMediaStore())
	tool.SetSendCallback(func(
		ctx context.Context,
		channel, chatID, content, replyToMessageID string,
		mediaParts []bus.MediaPart,
	) error {
		t.Fatal("send callback should not run for a directory path")
		return nil
	})

	ctx := WithToolContext(context.Background(), "telegram", "-1001")
	result := tool.Execute(ctx, map[string]any{"file_path": "."})
	if !result.IsError {
		t.Fatal("expected error for directory path")
	}
	if !strings.Contains(result.ForLLM, "expected a file") {
		t.Fatalf("expected 'expected a file' in error, got %q", result.ForLLM)
	}
}

func TestMessageTool_Execute_FileTooLarge(t *testing.T) {
	tool := NewMessageTool()
	dir := t.TempDir()
	bigPath := filepath.Join(dir, "big.bin")
	if err := os.WriteFile(bigPath, make([]byte, 1024), 0o644); err != nil {
		t.Fatalf("write big file: %v", err)
	}
	tool.ConfigureLocalMedia(dir, true, 512, []*regexp.Regexp{})
	tool.SetMediaStore(media.NewFileMediaStore())
	tool.SetSendCallback(func(
		ctx context.Context,
		channel, chatID, content, replyToMessageID string,
		mediaParts []bus.MediaPart,
	) error {
		t.Fatal("send callback should not run for an oversized file")
		return nil
	})

	ctx := WithToolContext(context.Background(), "telegram", "-1001")
	result := tool.Execute(ctx, map[string]any{"file_path": "big.bin"})
	if !result.IsError {
		t.Fatal("expected error for oversized file")
	}
	if !strings.Contains(result.ForLLM, "file too large") {
		t.Fatalf("expected 'file too large' in error, got %q", result.ForLLM)
	}
}

func TestMessageTool_ConfigureLocalMedia_DefaultMaxSize(t *testing.T) {
	tool := NewMessageTool()
	tool.ConfigureLocalMedia(t.TempDir(), true, 0, nil)
	if tool.maxFileSize != config.DefaultMaxMediaSize {
		t.Fatalf("maxFileSize = %d, want default %d", tool.maxFileSize, config.DefaultMaxMediaSize)
	}
}

func TestMessageTool_Execute_MediaStoreNotConfigured(t *testing.T) {
	tool := NewMessageTool()
	tool.ConfigureLocalMedia(t.TempDir(), true, 1024*1024, []*regexp.Regexp{})
	tool.SetSendCallback(func(
		ctx context.Context,
		channel, chatID, content, replyToMessageID string,
		mediaParts []bus.MediaPart,
	) error {
		t.Fatal("send callback should not run without a media store")
		return nil
	})

	ctx := WithToolContext(context.Background(), "telegram", "-1001")
	result := tool.Execute(ctx, map[string]any{"file_path": "anything.txt"})
	if !result.IsError {
		t.Fatal("expected error when media store is missing")
	}
	if !strings.Contains(result.ForLLM, "media store not configured") {
		t.Fatalf("expected 'media store not configured' in error, got %q", result.ForLLM)
	}
}

func TestMessageTool_Execute_MediaDeliveryNotConfigured(t *testing.T) {
	tool := NewMessageTool()
	// Media support enabled but no workspace: delivery is incomplete.
	tool.ConfigureLocalMedia("", true, 1024*1024, []*regexp.Regexp{})
	tool.SetMediaStore(media.NewFileMediaStore())
	tool.SetSendCallback(func(
		ctx context.Context,
		channel, chatID, content, replyToMessageID string,
		mediaParts []bus.MediaPart,
	) error {
		t.Fatal("send callback should not run without a workspace")
		return nil
	})

	ctx := WithToolContext(context.Background(), "telegram", "-1001")
	result := tool.Execute(ctx, map[string]any{"file_path": "anything.txt"})
	if !result.IsError {
		t.Fatal("expected error when workspace is not configured")
	}
	if !strings.Contains(result.ForLLM, "not configured") {
		t.Fatalf("expected 'not configured' in error, got %q", result.ForLLM)
	}
}

func TestDetectMessageMediaType_MagicBytes(t *testing.T) {
	dir := t.TempDir()
	pngPath := filepath.Join(dir, "photo")
	if err := os.WriteFile(pngPath, []byte("\x89PNG\r\n\x1a\n"), 0o644); err != nil {
		t.Fatalf("write png: %v", err)
	}
	if got := detectMessageMediaType(pngPath); got != "image/png" {
		t.Fatalf("detectMessageMediaType = %q, want image/png", got)
	}
}

func TestDetectMessageMediaType_FallbackToExtension(t *testing.T) {
	dir := t.TempDir()
	svgPath := filepath.Join(dir, "diagram.svg")
	if err := os.WriteFile(svgPath, []byte(`<svg></svg>`), 0o644); err != nil {
		t.Fatalf("write svg: %v", err)
	}
	if got := detectMessageMediaType(svgPath); got != "image/svg+xml" {
		t.Fatalf("detectMessageMediaType = %q, want image/svg+xml", got)
	}
}

func TestDetectMessageMediaType_UnknownFallsToOctetStream(t *testing.T) {
	dir := t.TempDir()
	blobPath := filepath.Join(dir, "blob")
	if err := os.WriteFile(blobPath, []byte{0x00, 0x01, 0x02, 0x03}, 0o644); err != nil {
		t.Fatalf("write blob: %v", err)
	}
	if got := detectMessageMediaType(blobPath); got != "application/octet-stream" {
		t.Fatalf("detectMessageMediaType = %q, want application/octet-stream", got)
	}
}
